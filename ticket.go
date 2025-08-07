package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FileSystem interface for mocking file operations
type FileSystem interface {
	UserHomeDir() (string, error)
	MkdirAll(path string, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadDir(dirname string) ([]os.DirEntry, error)
	Stat(name string) (os.FileInfo, error)
	Open(name string) (*os.File, error)
}

// RealFileSystem implements FileSystem using real file operations
type RealFileSystem struct{}

func (fs *RealFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *RealFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (fs *RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fs *RealFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	return os.ReadDir(dirname)
}

func (fs *RealFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *RealFileSystem) Open(name string) (*os.File, error) {
	return os.Open(name)
}

// Global variable for file system (can be mocked in tests)
var fileSystem FileSystem = &RealFileSystem{}

type Ticket struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// Implement list.Item interface
func (t Ticket) FilterValue() string {
	return t.Title + " " + t.URL
}

// Extract ticket number from URL
func (t Ticket) ExtractTicketNumber() string {
	// Look for Jira-style project keys with numbers (e.g., PROJ-123)
	re := regexp.MustCompile(`[A-Z]+-(\d+)`)
	matches := re.FindStringSubmatch(t.URL)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Look for numbers at the very end of URL paths (most specific first)
	// This regex ensures we get the last number in the path
	re = regexp.MustCompile(`.*[^/]/(\d+)(?:[/?#].*)?$`)
	matches = re.FindStringSubmatch(t.URL)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Fallback: any number at end of path
	re = regexp.MustCompile(`/(\d+)(?:[/?#].*)?$`)
	matches = re.FindStringSubmatch(t.URL)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Look for issue/ticket/task patterns with numbers
	re = regexp.MustCompile(`(?:issue|ticket|task)(?:s)?[/=](\d+)`)
	matches = re.FindStringSubmatch(strings.ToLower(t.URL))
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Look for any 6+ digit number
	re = regexp.MustCompile(`(\d{6,})`)
	matches = re.FindStringSubmatch(t.URL)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Default fallback
	return fmt.Sprintf("%06d", t.ID)
}

// For list display
func (t Ticket) GetTitle() string {
	ticketNum := t.ExtractTicketNumber()
	return fmt.Sprintf("SCR #%s - %s", ticketNum, t.Title)
}

func (t Ticket) GetDescription() string {
	return t.URL
}

type TicketStorage struct {
	Tickets []Ticket `json:"tickets"`
	NextID  int      `json:"next_id"`
}

type ImportResult struct {
	Added      int
	Duplicates int
	Errors     int
	ErrorLines []string
}

// createBackup creates a backup of the current tickets.json file
func createBackup() error {
	homeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		return err
	}
	
	dataDir := filepath.Join(homeDir, ".gotickets")
	filePath := filepath.Join(dataDir, "tickets.json")
	
	// Check if file exists
	if _, err := fileSystem.Stat(filePath); os.IsNotExist(err) {
		// No file to backup
		return nil
	}
	
	// Create backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(dataDir, fmt.Sprintf("tickets_backup_%s.json", timestamp))
	
	// Copy file to backup
	data, err := fileSystem.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read tickets file for backup: %v", err)
	}
	
	err = fileSystem.WriteFile(backupPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}
	
	return nil
}

func (ts *TicketStorage) AddTicket(title, url string) {
	// Create backup before adding ticket
	if err := createBackup(); err != nil {
		// Log error but continue with operation
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}
	
	ticket := Ticket{
		ID:        ts.NextID,
		Title:     title,
		URL:       url,
		CreatedAt: time.Now(),
	}
	ts.Tickets = append(ts.Tickets, ticket)
	ts.NextID++
}

func (ts *TicketStorage) Search(query string) []Ticket {
	if query == "" {
		return ts.Tickets
	}
	
	var results []Ticket
	query = strings.ToLower(query)
	
	for _, ticket := range ts.Tickets {
		if strings.Contains(strings.ToLower(ticket.Title), query) ||
		   strings.Contains(strings.ToLower(ticket.URL), query) {
			results = append(results, ticket)
		}
	}
	
	return results
}

func (ts *TicketStorage) DeleteTicket(id int) bool {
	// Create backup before deleting ticket
	if err := createBackup(); err != nil {
		// Log error but continue with operation
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}
	
	for i, ticket := range ts.Tickets {
		if ticket.ID == id {
			ts.Tickets = append(ts.Tickets[:i], ts.Tickets[i+1:]...)
			return true
		}
	}
	return false
}

func (ts *TicketStorage) HasTicketWithURL(url string) bool {
	for _, ticket := range ts.Tickets {
		if ticket.URL == url {
			return true
		}
	}
	return false
}

func (ts *TicketStorage) ImportFromFile(filePath string) (*ImportResult, error) {
	// Create backup before importing
	if err := createBackup(); err != nil {
		// Log error but continue with operation
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}
	
	result := &ImportResult{
		ErrorLines: make([]string, 0),
	}
	
	file, err := fileSystem.Open(filePath)
	if err != nil {
		return result, fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		// Пропускаем пустые строки
		if line == "" {
			continue
		}
		
		// Парсим формат "URL - Title"
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			result.Errors++
			result.ErrorLines = append(result.ErrorLines, fmt.Sprintf("Строка %d: неверный формат", lineNumber))
			continue
		}
		
		url := strings.TrimSpace(parts[0])
		title := strings.TrimSpace(parts[1])
		
		if url == "" || title == "" {
			result.Errors++
			result.ErrorLines = append(result.ErrorLines, fmt.Sprintf("Строка %d: пустая ссылка или название", lineNumber))
			continue
		}
		
		// Проверяем дубликаты
		if ts.HasTicketWithURL(url) {
			result.Duplicates++
			continue
		}
		
		// Добавляем тикет
		ts.AddTicket(title, url)
		result.Added++
	}
	
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("ошибка чтения файла: %v", err)
	}
	
	return result, nil
}

func (ts *TicketStorage) Save() error {
	homeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		return err
	}
	
	dataDir := filepath.Join(homeDir, ".gotickets")
	if err := fileSystem.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	
	filePath := filepath.Join(dataDir, "tickets.json")
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	
	return fileSystem.WriteFile(filePath, data, 0644)
}

func LoadTickets() (*TicketStorage, error) {
	homeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		return &TicketStorage{NextID: 1}, nil
	}
	
	filePath := filepath.Join(homeDir, ".gotickets", "tickets.json")
	return LoadTicketsFromPath(filePath)
}

// LoadTicketsFromPath loads tickets from a specific file path (useful for testing)
func LoadTicketsFromPath(filePath string) (*TicketStorage, error) {
	data, err := fileSystem.ReadFile(filePath)
	if err != nil {
		return &TicketStorage{NextID: 1}, nil
	}
	
	var storage TicketStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return &TicketStorage{NextID: 1}, nil
	}
	
	if storage.NextID == 0 {
		storage.NextID = 1
	}
	
	return &storage, nil
}

// listBackups returns a list of available backup files
func listBackups() ([]string, error) {
	homeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	dataDir := filepath.Join(homeDir, ".gotickets")
	
	// Read directory contents
	entries, err := fileSystem.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}
	
	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "tickets_backup_") && strings.HasSuffix(entry.Name(), ".json") {
			backups = append(backups, entry.Name())
		}
	}
	
	return backups, nil
}

// restoreFromBackup restores tickets from a backup file
func restoreFromBackup(backupName string) error {
	homeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		return err
	}
	
	dataDir := filepath.Join(homeDir, ".gotickets")
	backupPath := filepath.Join(dataDir, backupName)
	filePath := filepath.Join(dataDir, "tickets.json")
	
	// Check if backup file exists
	if _, err := fileSystem.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupName)
	}
	
	// Read backup file
	data, err := fileSystem.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}
	
	// Validate JSON
	var storage TicketStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return fmt.Errorf("backup file is corrupted: %v", err)
	}
	
	// Restore the file
	err = fileSystem.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to restore from backup: %v", err)
	}
	
	return nil
}