package storage

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

// FileSystem abstracts filesystem interactions for easier testing
type FileSystem interface {
	UserHomeDir() (string, error)
	MkdirAll(path string, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadDir(dirname string) ([]os.DirEntry, error)
	Stat(name string) (os.FileInfo, error)
	Open(name string) (*os.File, error)
}

// RealFileSystem implements FileSystem using the os package
type RealFileSystem struct{}

func (fs *RealFileSystem) UserHomeDir() (string, error) { return os.UserHomeDir() }
func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (fs *RealFileSystem) ReadFile(filename string) ([]byte, error) { return os.ReadFile(filename) }
func (fs *RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
func (fs *RealFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) { return os.ReadDir(dirname) }
func (fs *RealFileSystem) Stat(name string) (os.FileInfo, error)         { return os.Stat(name) }
func (fs *RealFileSystem) Open(name string) (*os.File, error)            { return os.Open(name) }

type Ticket struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// FilterValue implements bubbles list.Item interface
func (t Ticket) FilterValue() string { return t.Title + " " + t.URL }

// ExtractTicketNumber tries multiple strategies to get a ticket number from URL
func (t Ticket) ExtractTicketNumber() string {
	re := regexp.MustCompile(`[A-Z]+-(\d+)`)
	if matches := re.FindStringSubmatch(t.URL); len(matches) > 1 {
		return matches[1]
	}
	re = regexp.MustCompile(`.*[^/]/(\d+)(?:[/?#].*)?$`)
	if matches := re.FindStringSubmatch(t.URL); len(matches) > 1 {
		return matches[1]
	}
	re = regexp.MustCompile(`/(\d+)(?:[/?#].*)?$`)
	if matches := re.FindStringSubmatch(t.URL); len(matches) > 1 {
		return matches[1]
	}
	re = regexp.MustCompile(`(?:issue|ticket|task)(?:s)?[/=](\d+)`)
	if matches := re.FindStringSubmatch(strings.ToLower(t.URL)); len(matches) > 1 {
		return matches[1]
	}
	re = regexp.MustCompile(`(\d{6,})`)
	if matches := re.FindStringSubmatch(t.URL); len(matches) > 1 {
		return matches[1]
	}
	return fmt.Sprintf("%06d", t.ID)
}

func (t Ticket) GetTitle() string {
	return fmt.Sprintf("SCR #%s - %s", t.ExtractTicketNumber(), t.Title)
}
func (t Ticket) GetDescription() string { return t.URL }

type TicketStorage struct {
	Tickets []Ticket `json:"tickets"`
	NextID  int      `json:"next_id"`
	fs      FileSystem
}

type ImportResult struct {
	Added      int
	Duplicates int
	Errors     int
	ErrorLines []string
}

func NewTicketStorage(fs FileSystem) *TicketStorage { return &TicketStorage{NextID: 1, fs: fs} }

func (ts *TicketStorage) getFS() FileSystem {
	if ts == nil || ts.fs == nil {
		return &RealFileSystem{}
	}
	return ts.fs
}

func CreateBackupUsing(fs FileSystem) error {
	homeDir, err := fs.UserHomeDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(homeDir, ".gotickets")
	filePath := filepath.Join(dataDir, "tickets.json")
	if _, err := fs.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(dataDir, fmt.Sprintf("tickets_backup_%s.json", timestamp))
	data, err := fs.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read tickets file for backup: %v", err)
	}
	if err := fs.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}
	return nil
}

func (ts *TicketStorage) AddTicket(title, url string) {
	if err := CreateBackupUsing(ts.getFS()); err != nil {
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}
	ticket := Ticket{ID: ts.NextID, Title: title, URL: url, CreatedAt: time.Now()}
	ts.Tickets = append(ts.Tickets, ticket)
	ts.NextID++
}

func (ts *TicketStorage) Search(query string) []Ticket {
	if query == "" {
		return ts.Tickets
	}
	var results []Ticket
	q := strings.ToLower(query)
	for _, ticket := range ts.Tickets {
		if strings.Contains(strings.ToLower(ticket.Title), q) || strings.Contains(strings.ToLower(ticket.URL), q) {
			results = append(results, ticket)
		}
	}
	return results
}

func (ts *TicketStorage) DeleteTicket(id int) bool {
	if err := CreateBackupUsing(ts.getFS()); err != nil {
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
	if err := CreateBackupUsing(ts.getFS()); err != nil {
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}
	result := &ImportResult{ErrorLines: make([]string, 0)}
	f, err := ts.getFS().Open(filePath)
	if err != nil {
		return result, fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
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
		if ts.HasTicketWithURL(url) {
			result.Duplicates++
			continue
		}
		ts.AddTicket(title, url)
		result.Added++
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("ошибка чтения файла: %v", err)
	}
	return result, nil
}

func (ts *TicketStorage) Save() error {
	fs := ts.getFS()
	homeDir, err := fs.UserHomeDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(homeDir, ".gotickets")
	if err := fs.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(dataDir, "tickets.json")
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	return fs.WriteFile(filePath, data, 0644)
}

func LoadTicketsWithFS(fs FileSystem) (*TicketStorage, error) {
	homeDir, err := fs.UserHomeDir()
	if err != nil {
		return &TicketStorage{NextID: 1, fs: fs}, nil
	}
	filePath := filepath.Join(homeDir, ".gotickets", "tickets.json")
	return LoadTicketsFromPathWithFS(fs, filePath)
}

func LoadTicketsFromPathWithFS(fs FileSystem, filePath string) (*TicketStorage, error) {
	data, err := fs.ReadFile(filePath)
	if err != nil {
		return &TicketStorage{NextID: 1, fs: fs}, nil
	}
	var storage TicketStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		storage = TicketStorage{NextID: 1}
		storage.fs = fs
		return &storage, nil
	}
	if storage.NextID == 0 {
		storage.NextID = 1
	}
	storage.fs = fs
	return &storage, nil
}

func ListBackupsUsing(fs FileSystem) ([]string, error) {
	homeDir, err := fs.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(homeDir, ".gotickets")
	entries, err := fs.ReadDir(dataDir)
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

func RestoreFromBackupUsing(fs FileSystem, backupName string) error {
	homeDir, err := fs.UserHomeDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(homeDir, ".gotickets")
	backupPath := filepath.Join(dataDir, backupName)
	filePath := filepath.Join(dataDir, "tickets.json")
	if _, err := fs.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupName)
	}
	data, err := fs.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}
	var storage TicketStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return fmt.Errorf("backup file is corrupted: %v", err)
	}
	if err := fs.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore from backup: %v", err)
	}
	return nil
}
