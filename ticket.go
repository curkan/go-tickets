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

func (ts *TicketStorage) AddTicket(title, url string) {
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
	result := &ImportResult{
		ErrorLines: make([]string, 0),
	}
	
	file, err := os.Open(filePath)
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	dataDir := filepath.Join(homeDir, ".gotickets")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	
	filePath := filepath.Join(dataDir, "tickets.json")
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, 0644)
}

func LoadTickets() (*TicketStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &TicketStorage{NextID: 1}, nil
	}
	
	filePath := filepath.Join(homeDir, ".gotickets", "tickets.json")
	data, err := os.ReadFile(filePath)
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