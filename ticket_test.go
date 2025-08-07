package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)



// MockFileSystem for testing
type MockFileSystem struct {
	homeDir     string
	files       map[string][]byte
	directories map[string]bool
	statResults map[string]os.FileInfo
	errors      map[string]error
}

func NewMockFileSystem(homeDir string) *MockFileSystem {
	return &MockFileSystem{
		homeDir:     homeDir,
		files:       make(map[string][]byte),
		directories: make(map[string]bool),
		statResults: make(map[string]os.FileInfo),
		errors:      make(map[string]error),
	}
}

func (fs *MockFileSystem) UserHomeDir() (string, error) {
	if err, exists := fs.errors["UserHomeDir"]; exists {
		return "", err
	}
	return fs.homeDir, nil
}

func (fs *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err, exists := fs.errors["MkdirAll"]; exists {
		return err
	}
	fs.directories[path] = true
	return nil
}

func (fs *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if err, exists := fs.errors["ReadFile"]; exists {
		return nil, err
	}
	if data, exists := fs.files[filename]; exists {
		return data, nil
	}
	return nil, &os.PathError{Op: "read", Path: filename, Err: os.ErrNotExist}
}

func (fs *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err, exists := fs.errors["WriteFile"]; exists {
		return err
	}
	fs.files[filename] = data
	return nil
}

func (fs *MockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	if err, exists := fs.errors["ReadDir"]; exists {
		return nil, err
	}
	
	var entries []os.DirEntry
	for filename := range fs.files {
		if strings.HasPrefix(filename, dirname) {
			// Create a simple mock DirEntry
			entry := &mockDirEntry{name: filepath.Base(filename)}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (fs *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if err, exists := fs.errors["Stat"]; exists {
		return nil, err
	}
	if _, exists := fs.files[name]; exists {
		return &mockFileInfo{name: filepath.Base(name)}, nil
	}
	return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
}

func (fs *MockFileSystem) Open(name string) (*os.File, error) {
	if err, exists := fs.errors["Open"]; exists {
		return nil, err
	}
	if data, exists := fs.files[name]; exists {
		// Create a temporary file with the data
		tmpFile, err := os.CreateTemp("", "mock_file_*")
		if err != nil {
			return nil, err
		}
		_, err = tmpFile.Write(data)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, err
		}
		tmpFile.Seek(0, 0)
		return tmpFile, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

// Mock implementations
type mockDirEntry struct {
	name string
}

func (d *mockDirEntry) Name() string               { return d.name }
func (d *mockDirEntry) IsDir() bool               { return false }
func (d *mockDirEntry) Type() os.FileMode         { return 0 }
func (d *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

type mockFileInfo struct {
	name string
}

func (f *mockFileInfo) Name() string       { return f.name }
func (f *mockFileInfo) Size() int64        { return 0 }
func (f *mockFileInfo) Mode() os.FileMode  { return 0 }
func (f *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (f *mockFileInfo) IsDir() bool        { return false }
func (f *mockFileInfo) Sys() interface{}   { return nil }



// Mock functions for testing
func createBackupForTest() error {
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

func listBackupsForTest() ([]string, error) {
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

func restoreFromBackupForTest(backupName string) error {
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

func TestTicketStorage_AddTicket(t *testing.T) {
	ts := &TicketStorage{NextID: 1}
	
	title := "Test Ticket"
	url := "https://example.com"
	
	ts.AddTicket(title, url)
	
	if len(ts.Tickets) != 1 {
		t.Errorf("Expected 1 ticket, got %d", len(ts.Tickets))
	}
	
	ticket := ts.Tickets[0]
	if ticket.ID != 1 {
		t.Errorf("Expected ID 1, got %d", ticket.ID)
	}
	if ticket.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, ticket.Title)
	}
	if ticket.URL != url {
		t.Errorf("Expected URL '%s', got '%s'", url, ticket.URL)
	}
	if ts.NextID != 2 {
		t.Errorf("Expected NextID 2, got %d", ts.NextID)
	}
}

func TestTicketStorage_Search(t *testing.T) {
	ts := &TicketStorage{NextID: 1}
	
	ts.AddTicket("Bug Report", "https://github.com/issue/1")
	ts.AddTicket("Feature Request", "https://example.com/feature")
	ts.AddTicket("Task", "https://docs.example.com")
	
	tests := []struct {
		query    string
		expected int
		desc     string
	}{
		{"", 3, "empty query should return all tickets"},
		{"bug", 1, "should find tickets with 'bug' in title or URL"},
		{"BUG", 1, "search should be case insensitive"},
		{"github", 1, "should find tickets with 'github' in URL"},
		{"feature", 1, "should find tickets with 'feature' in URL or title"},
		{"xyz", 0, "should return no results for non-matching query"},
		{"docs", 1, "should find tickets with 'docs' in URL"},
	}
	
	for _, test := range tests {
		results := ts.Search(test.query)
		if len(results) != test.expected {
			t.Errorf("Query '%s': %s. Expected %d results, got %d", 
				test.query, test.desc, test.expected, len(results))
		}
	}
}

func TestTicketStorage_DeleteTicket(t *testing.T) {
	ts := &TicketStorage{NextID: 1}
	
	ts.AddTicket("Test Ticket 1", "https://example.com/1")
	ts.AddTicket("Test Ticket 2", "https://example.com/2")
	ts.AddTicket("Test Ticket 3", "https://example.com/3")
	
	initialCount := len(ts.Tickets)
	if initialCount != 3 {
		t.Fatalf("Expected 3 tickets initially, got %d", initialCount)
	}
	
	// Delete existing ticket
	deleted := ts.DeleteTicket(2)
	if !deleted {
		t.Error("Expected DeleteTicket to return true for existing ticket")
	}
	
	if len(ts.Tickets) != 2 {
		t.Errorf("Expected 2 tickets after deletion, got %d", len(ts.Tickets))
	}
	
	// Verify the correct ticket was deleted
	for _, ticket := range ts.Tickets {
		if ticket.ID == 2 {
			t.Error("Ticket with ID 2 should have been deleted")
		}
	}
	
	// Try to delete non-existing ticket
	deleted = ts.DeleteTicket(999)
	if deleted {
		t.Error("Expected DeleteTicket to return false for non-existing ticket")
	}
	
	if len(ts.Tickets) != 2 {
		t.Errorf("Expected 2 tickets after failed deletion, got %d", len(ts.Tickets))
	}
}

func TestTicketStorage_SaveAndLoad(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create test storage
	ts := &TicketStorage{NextID: 1}
	ts.AddTicket("Test Ticket 1", "https://example.com/1")
	ts.AddTicket("Test Ticket 2", "https://example.com/2")
	
	// Save the storage
	err := ts.Save()
	if err != nil {
		t.Fatalf("Failed to save tickets: %v", err)
	}
	
	// Verify file was created
	filePath := filepath.Join(tempDir, ".gotickets", "tickets.json")
	if _, err := mockFS.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Tickets file was not created")
	}
	
	// Load the storage using the specific path
	loaded, err := LoadTicketsFromPath(filePath)
	if err != nil {
		t.Fatalf("Failed to load tickets: %v", err)
	}
	
	// Verify loaded data
	if len(loaded.Tickets) != 2 {
		t.Errorf("Expected 2 loaded tickets, got %d", len(loaded.Tickets))
	}
	if loaded.NextID != 3 {
		t.Errorf("Expected NextID 3, got %d", loaded.NextID)
	}
	if loaded.Tickets[0].Title != "Test Ticket 1" {
		t.Errorf("Expected first ticket title 'Test Ticket 1', got '%s'", loaded.Tickets[0].Title)
	}
}

func TestLoadTickets_NonExistentFile(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Load tickets from non-existent file using specific path
	filePath := filepath.Join(tempDir, ".gotickets", "tickets.json")
	loaded, err := LoadTicketsFromPath(filePath)
	if err != nil {
		t.Fatalf("LoadTickets should not return error for non-existent file: %v", err)
	}
	
	if len(loaded.Tickets) != 0 {
		t.Errorf("Expected 0 tickets for new storage, got %d", len(loaded.Tickets))
	}
	if loaded.NextID != 1 {
		t.Errorf("Expected NextID 1 for new storage, got %d", loaded.NextID)
	}
}

func TestLoadTickets_CorruptedFile(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Write corrupted JSON to the test data directory
	dataDir := filepath.Join(tempDir, ".gotickets")
	
	filePath := filepath.Join(dataDir, "tickets.json")
	err := mockFS.WriteFile(filePath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}
	
	// Load tickets from corrupted file using specific path
	loaded, err := LoadTicketsFromPath(filePath)
	if err != nil {
		t.Fatalf("LoadTickets should not return error for corrupted file: %v", err)
	}
	
	if len(loaded.Tickets) != 0 {
		t.Errorf("Expected 0 tickets for corrupted file, got %d", len(loaded.Tickets))
	}
	if loaded.NextID != 1 {
		t.Errorf("Expected NextID 1 for corrupted file, got %d", loaded.NextID)
	}
}

func TestTicket_JSONSerialization(t *testing.T) {
	ticket := Ticket{
		ID:        1,
		Title:     "Test Ticket",
		URL:       "https://example.com",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	// Marshal to JSON
	data, err := json.Marshal(ticket)
	if err != nil {
		t.Fatalf("Failed to marshal ticket: %v", err)
	}
	
	// Unmarshal from JSON
	var unmarshaled Ticket
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ticket: %v", err)
	}
	
	// Verify fields
	if unmarshaled.ID != ticket.ID {
		t.Errorf("ID mismatch: expected %d, got %d", ticket.ID, unmarshaled.ID)
	}
	if unmarshaled.Title != ticket.Title {
		t.Errorf("Title mismatch: expected '%s', got '%s'", ticket.Title, unmarshaled.Title)
	}
	if unmarshaled.URL != ticket.URL {
		t.Errorf("URL mismatch: expected '%s', got '%s'", ticket.URL, unmarshaled.URL)
	}
	if !unmarshaled.CreatedAt.Equal(ticket.CreatedAt) {
		t.Errorf("CreatedAt mismatch: expected %v, got %v", ticket.CreatedAt, unmarshaled.CreatedAt)
	}
}

func TestTicketStorage_HasTicketWithURL(t *testing.T) {
	ts := &TicketStorage{NextID: 1}
	ts.AddTicket("Test Ticket", "https://example.com")
	
	// Test existing URL
	if !ts.HasTicketWithURL("https://example.com") {
		t.Error("Expected HasTicketWithURL to return true for existing URL")
	}
	
	// Test non-existing URL
	if ts.HasTicketWithURL("https://nonexistent.com") {
		t.Error("Expected HasTicketWithURL to return false for non-existing URL")
	}
	
	// Test empty URL
	if ts.HasTicketWithURL("") {
		t.Error("Expected HasTicketWithURL to return false for empty URL")
	}
}

func TestTicketStorage_ImportFromFile(t *testing.T) {
	// Create a temporary file with test data
	tempFile := filepath.Join(t.TempDir(), "test_import.txt")
	testData := `https://example.com/1 - First Ticket
https://example.com/2 - Second Ticket
https://example.com/3 - Third Ticket
https://example.com/1 - Duplicate First Ticket
invalid line without dash
 - Empty URL
https://example.com/4 - `
	
	err := os.WriteFile(tempFile, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	ts := &TicketStorage{NextID: 1}
	
	// Import from file
	result, err := ts.ImportFromFile(tempFile)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	
	// Check results
	if result.Added != 3 {
		t.Errorf("Expected 3 tickets added, got %d", result.Added)
	}
	if result.Duplicates != 1 {
		t.Errorf("Expected 1 duplicate, got %d", result.Duplicates)
	}
	if result.Errors != 3 {
		t.Errorf("Expected 3 errors, got %d", result.Errors)
	}
	if len(result.ErrorLines) != 3 {
		t.Errorf("Expected 3 error lines, got %d", len(result.ErrorLines))
	}
	
	// Check that tickets were actually added
	if len(ts.Tickets) != 3 {
		t.Errorf("Expected 3 tickets in storage, got %d", len(ts.Tickets))
	}
	
	// Check ticket content
	expectedTickets := []struct {
		url   string
		title string
	}{
		{"https://example.com/1", "First Ticket"},
		{"https://example.com/2", "Second Ticket"},
		{"https://example.com/3", "Third Ticket"},
	}
	
	for i, expected := range expectedTickets {
		if i >= len(ts.Tickets) {
			t.Errorf("Missing ticket %d", i)
			continue
		}
		
		ticket := ts.Tickets[i]
		if ticket.URL != expected.url {
			t.Errorf("Ticket %d URL: expected %s, got %s", i, expected.url, ticket.URL)
		}
		if ticket.Title != expected.title {
			t.Errorf("Ticket %d Title: expected %s, got %s", i, expected.title, ticket.Title)
		}
	}
}

func TestTicketStorage_ImportFromFile_NonExistentFile(t *testing.T) {
	ts := &TicketStorage{NextID: 1}
	
	result, err := ts.ImportFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if result.Added != 0 || result.Duplicates != 0 || result.Errors != 0 {
		t.Error("Expected empty result for failed import")
	}
}

func TestTicketStorage_ImportFromFile_EmptyFile(t *testing.T) {
	// Create an empty temporary file
	tempFile := filepath.Join(t.TempDir(), "empty.txt")
	err := os.WriteFile(tempFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}
	
	ts := &TicketStorage{NextID: 1}
	
	result, err := ts.ImportFromFile(tempFile)
	if err != nil {
		t.Errorf("Should not error on empty file: %v", err)
	}
	
	if result.Added != 0 || result.Duplicates != 0 || result.Errors != 0 {
		t.Error("Expected all zeros for empty file import")
	}
}

func TestTicket_ExtractTicketNumber(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub issue URL",
			url:      "https://github.com/user/repo/issues/123",
			expected: "123",
		},
		{
			name:     "GitHub issue URL with trailing slash",
			url:      "https://github.com/user/repo/issues/456/",
			expected: "456",
		},
		{
			name:     "Jira-style URL",
			url:      "https://company.atlassian.net/browse/PROJ-789",
			expected: "789",
		},
		{
			name:     "Tracker URL with issue number",
			url:      "https://tracker.example.com/issues/424846",
			expected: "424846",
		},
		{
			name:     "URL with task parameter",
			url:      "https://example.com/task=999888",
			expected: "999888",
		},
		{
			name:     "URL with ticket in path",
			url:      "https://example.com/tickets/111222",
			expected: "111222",
		},
		{
			name:     "Complex URL with multiple numbers",
			url:      "https://example.com/project/123/issues/456789",
			expected: "456789", // Should pick the last number
		},
		{
			name:     "URL without clear ticket number",
			url:      "https://example.com/some/path",
			expected: "000001", // Falls back to ticket ID (1) padded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket := Ticket{
				ID:    1,
				Title: "Test Ticket",
				URL:   tt.url,
			}
			result := ticket.ExtractTicketNumber()
			if result != tt.expected {
				t.Errorf("ExtractTicketNumber() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTicket_GetTitle(t *testing.T) {
	ticket := Ticket{
		ID:    1,
		Title: "Test Issue",
		URL:   "https://github.com/user/repo/issues/12345",
	}
	
	expected := "SCR #12345 - Test Issue"
	result := ticket.GetTitle()
	
	if result != expected {
		t.Errorf("GetTitle() = %v, want %v", result, expected)
	}
}

func TestCreateBackup(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create test storage and save it
	ts := &TicketStorage{NextID: 1}
	ts.AddTicket("Test Ticket", "https://example.com")
	err := ts.Save()
	if err != nil {
		t.Fatalf("Failed to save test tickets: %v", err)
	}
	
	// Create backup using test function
	err = createBackupForTest()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	
	// Check that backup file exists
	dataDir := filepath.Join(tempDir, ".gotickets")
	entries, err := mockFS.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("Failed to read data directory: %v", err)
	}
	
	backupFound := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "tickets_backup_") && strings.HasSuffix(entry.Name(), ".json") {
			backupFound = true
			break
		}
	}
	
	if !backupFound {
		t.Error("Backup file was not created")
	}
}

func TestListBackups(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create some backup files
	dataDir := filepath.Join(tempDir, ".gotickets")
	
	// Create test backup files
	testBackups := []string{
		"tickets_backup_2023-01-01_12-00-00.json",
		"tickets_backup_2023-01-02_12-00-00.json",
	}
	
	for _, backup := range testBackups {
		backupPath := filepath.Join(dataDir, backup)
		err := mockFS.WriteFile(backupPath, []byte(`{"tickets":[],"next_id":1}`), 0644)
		if err != nil {
			t.Fatalf("Failed to create test backup file: %v", err)
		}
	}
	
	// List backups using test function
	backups, err := listBackupsForTest()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}
	
	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}
	
	// Check that all test backups are found
	for _, testBackup := range testBackups {
		found := false
		for _, backup := range backups {
			if backup == testBackup {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected backup %s not found", testBackup)
		}
	}
}

func TestRestoreFromBackup(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create test backup file
	dataDir := filepath.Join(tempDir, ".gotickets")
	
	backupData := `{
		"tickets": [
			{
				"id": 1,
				"title": "Restored Ticket",
				"url": "https://example.com/restored",
				"created_at": "2023-01-01T12:00:00Z"
			}
		],
		"next_id": 2
	}`
	
	backupName := "tickets_backup_2023-01-01_12-00-00.json"
	backupPath := filepath.Join(dataDir, backupName)
	err := mockFS.WriteFile(backupPath, []byte(backupData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test backup file: %v", err)
	}
	
	// Restore from backup using test function
	err = restoreFromBackupForTest(backupName)
	if err != nil {
		t.Fatalf("Failed to restore from backup: %v", err)
	}
	
	// Load tickets and verify using specific path
	filePath := filepath.Join(dataDir, "tickets.json")
	loaded, err := LoadTicketsFromPath(filePath)
	if err != nil {
		t.Fatalf("Failed to load tickets after restore: %v", err)
	}
	
	if len(loaded.Tickets) != 1 {
		t.Errorf("Expected 1 ticket after restore, got %d", len(loaded.Tickets))
	}
	
	if loaded.Tickets[0].Title != "Restored Ticket" {
		t.Errorf("Expected ticket title 'Restored Ticket', got '%s'", loaded.Tickets[0].Title)
	}
	
	if loaded.NextID != 2 {
		t.Errorf("Expected NextID 2 after restore, got %d", loaded.NextID)
	}
}

func TestRestoreFromBackup_NonExistent(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Try to restore from non-existent backup
	err := restoreFromBackupForTest("non_existent_backup.json")
	if err == nil {
		t.Error("Expected error when restoring from non-existent backup")
	}
}

func TestRestoreFromBackup_Corrupted(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create corrupted backup file
	dataDir := filepath.Join(tempDir, ".gotickets")
	
	backupName := "tickets_backup_2023-01-01_12-00-00.json"
	backupPath := filepath.Join(dataDir, backupName)
	err := mockFS.WriteFile(backupPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted backup file: %v", err)
	}
	
	// Try to restore from corrupted backup
	err = restoreFromBackupForTest(backupName)
	if err == nil {
		t.Error("Expected error when restoring from corrupted backup")
	}
}

func TestIsolationFromRealEnvironment(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Verify that we're using a temporary directory
	if !strings.Contains(tempDir, "TestIsolationFromRealEnvironment") {
		t.Errorf("Expected test to use temporary directory, got: %s", tempDir)
	}
	
	// Verify that fileSystem.UserHomeDir returns our test directory
	homeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		t.Fatalf("fileSystem.UserHomeDir failed: %v", err)
	}
	
	if homeDir != tempDir {
		t.Errorf("fileSystem.UserHomeDir returned %s, expected %s", homeDir, tempDir)
	}
	
	// Verify that LoadTickets uses our test directory
	storage, err := LoadTickets()
	if err != nil {
		t.Fatalf("LoadTickets failed: %v", err)
	}
	
	// Add a ticket and save it
	storage.AddTicket("Test Isolation", "https://example.com/test")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}
	
	// Verify the file was created in our test directory
	expectedPath := filepath.Join(tempDir, ".gotickets", "tickets.json")
	if _, err := mockFS.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file to be created at %s", expectedPath)
	}
	
	// Verify that our mock file system is working correctly
	// by checking that the file exists in our mock filesystem
	// We don't need to check the real filesystem as it could interfere with real data
	t.Logf("Test completed successfully using mock filesystem in: %s", tempDir)
}