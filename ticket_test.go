package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Save the original home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	
	// Set temporary home directory
	os.Setenv("HOME", tempDir)
	
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
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Tickets file was not created")
	}
	
	// Load the storage
	loaded, err := LoadTickets()
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
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Save the original home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	
	// Set temporary home directory with no existing tickets
	os.Setenv("HOME", tempDir)
	
	// Load tickets from non-existent file
	loaded, err := LoadTickets()
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
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Save the original home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	
	// Set temporary home directory
	os.Setenv("HOME", tempDir)
	
	// Create directory and write corrupted JSON
	dataDir := filepath.Join(tempDir, ".gotickets")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}
	
	filePath := filepath.Join(dataDir, "tickets.json")
	err = os.WriteFile(filePath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}
	
	// Load tickets from corrupted file
	loaded, err := LoadTickets()
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