package unit

import (
	"path/filepath"
	"strings"
	"testing"
	
	"gotickets/internal/storage"
	"gotickets/test/mocks"
)

func TestTicketStorage_AddTicket(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)
	
	ticketStorage := storage.NewTicketStorage(mockFS)
	
	ticketStorage.AddTicket("Test Ticket", "https://example.com")
	
	if len(ticketStorage.Tickets) != 1 {
		t.Fatalf("Expected 1 ticket, got %d", len(ticketStorage.Tickets))
	}
	
	ticket := ticketStorage.Tickets[0]
	if ticket.Title != "Test Ticket" || ticket.URL != "https://example.com" {
		t.Fatalf("Unexpected ticket data: %+v", ticket)
	}
	
	if ticketStorage.NextID != 2 {
		t.Fatalf("Expected NextID to be 2, got %d", ticketStorage.NextID)
	}
}

func TestTicketStorage_Search(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)
	
	ticketStorage := storage.NewTicketStorage(mockFS)
	
	ticketStorage.AddTicket("Bug Fix", "https://example.com/bug")
	ticketStorage.AddTicket("Feature Request", "https://example.com/feature") 
	ticketStorage.AddTicket("Documentation", "https://docs.example.com")
	
	// Search by title
	results := ticketStorage.Search("bug")
	if len(results) != 1 || results[0].Title != "Bug Fix" {
		t.Fatalf("Expected 1 result for 'bug', got %d", len(results))
	}
	
	// Search by URL
	results = ticketStorage.Search("docs")
	if len(results) != 1 || results[0].Title != "Documentation" {
		t.Fatalf("Expected 1 result for 'docs', got %d", len(results))
	}
	
	// Empty search should return all
	results = ticketStorage.Search("")
	if len(results) != 3 {
		t.Fatalf("Expected 3 results for empty search, got %d", len(results))
	}
}

func TestTicketStorage_DeleteTicket(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)
	
	ticketStorage := storage.NewTicketStorage(mockFS)
	
	ticketStorage.AddTicket("Test 1", "https://example.com/1")
	ticketStorage.AddTicket("Test 2", "https://example.com/2")
	
	// Delete first ticket (ID 1)
	if !ticketStorage.DeleteTicket(1) {
		t.Fatal("Expected successful deletion of ticket ID 1")
	}
	
	if len(ticketStorage.Tickets) != 1 {
		t.Fatalf("Expected 1 ticket after deletion, got %d", len(ticketStorage.Tickets))
	}
	
	if ticketStorage.Tickets[0].ID != 2 {
		t.Fatalf("Expected remaining ticket to have ID 2, got %d", ticketStorage.Tickets[0].ID)
	}
	
	// Try to delete non-existent ticket
	if ticketStorage.DeleteTicket(999) {
		t.Fatal("Expected deletion of non-existent ticket to fail")
	}
}

func TestTicketStorage_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)
	
	// Create and populate storage
	storage1 := storage.NewTicketStorage(mockFS)
	storage1.AddTicket("Test Ticket", "https://example.com")
	
	// Save
	if err := storage1.Save(); err != nil {
		t.Fatalf("Failed to save storage: %v", err)
	}
	
	// Load
	storage2, err := storage.LoadTicketsWithFS(mockFS)
	if err != nil {
		t.Fatalf("Failed to load storage: %v", err)
	}
	
	if len(storage2.Tickets) != 1 {
		t.Fatalf("Expected 1 ticket after load, got %d", len(storage2.Tickets))
	}
	
	if storage2.Tickets[0].Title != "Test Ticket" {
		t.Fatalf("Unexpected ticket title after load: %s", storage2.Tickets[0].Title)
	}
	
	if storage2.NextID != 2 {
		t.Fatalf("Expected NextID to be 2 after load, got %d", storage2.NextID)
	}
}

func TestTicket_ExtractTicketNumber(t *testing.T) {
	testCases := []struct {
		url      string
		expected string
	}{
		{"https://jira.example.com/PROJ-123", "123"},
		{"https://github.com/user/repo/issues/456", "456"},
		{"https://example.com/ticket/789", "789"},
		{"https://example.com/tasks/task=999", "999"},
		{"https://example.com/no-numbers", "000001"}, // Falls back to ID
	}
	
	for _, tc := range testCases {
		ticket := storage.Ticket{ID: 1, Title: "Test", URL: tc.url}
		result := ticket.ExtractTicketNumber()
		if result != tc.expected {
			t.Errorf("ExtractTicketNumber(%s) = %s, want %s", tc.url, result, tc.expected)
		}
	}
}

func TestCreateBackup_NoExistingTicketsFile(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)

	// No tickets.json exists → should be no-op without error
	if err := storage.CreateBackupUsing(mockFS); err != nil {
		t.Fatalf("CreateBackupUsing() returned error for non-existent file: %v", err)
	}
}

func TestCreateBackup_ReadFileError(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)

	dataDir := filepath.Join(tempDir, ".gotickets")
	ticketsPath := filepath.Join(dataDir, "tickets.json")

	// Seed a tickets.json file
	if err := mockFS.WriteFile(ticketsPath, []byte(`{"tickets":[],"next_id":1}`), 0644); err != nil {
		t.Fatalf("failed to seed tickets.json: %v", err)
	}
	// Force ReadFile error
	mockFS.SetError("ReadFile", mocks.AssertErr("read failure"))

	if err := storage.CreateBackupUsing(mockFS); err == nil || !strings.Contains(err.Error(), "failed to read tickets file for backup") {
		t.Fatalf("expected read error from createBackup, got: %v", err)
	}
}

func TestListBackups_ErrorFromReadDir(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)

	mockFS.SetError("ReadDir", mocks.AssertErr("boom"))
	if _, err := storage.ListBackupsUsing(mockFS); err == nil {
		t.Fatal("expected error from listBackups when ReadDir fails")
	}
}

func TestLoadTickets_UserHomeDirError(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)

	mockFS.SetError("UserHomeDir", mocks.AssertErr("cannot resolve home"))
	ticketStorage, err := storage.LoadTicketsWithFS(mockFS)
	if err != nil {
		t.Fatalf("LoadTicketsWithFS should not return error on UserHomeDir failure: %v", err)
	}
	if ticketStorage == nil || ticketStorage.NextID != 1 || len(ticketStorage.Tickets) != 0 {
		t.Fatalf("unexpected storage returned: %+v", ticketStorage)
	}
}

func TestStorage_ImportFromFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)

	// Prepare import file inside mock FS
	importPath := filepath.Join(tempDir, "import.txt")
	if err := mockFS.WriteFile(importPath, []byte("https://example.com/42 - The Answer\n"), 0644); err != nil {
		t.Fatalf("failed to write import file: %v", err)
	}

	// Test storage import functionality directly
	ticketStorage := storage.NewTicketStorage(mockFS)
	result, err := ticketStorage.ImportFromFile(importPath)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}
	
	if result.Added != 1 || result.Errors != 0 {
		t.Fatalf("unexpected importResult: %+v", result)
	}
	if len(ticketStorage.Tickets) != 1 {
		t.Fatalf("expected 1 ticket after import, got %d", len(ticketStorage.Tickets))
	}
}

func TestStorage_HasTicketWithURL_DuplicateDetection(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := mocks.NewMockFileSystem(tempDir)

	// Create a ticket storage with the mock filesystem
	ticketStorage := storage.NewTicketStorage(mockFS)

	// Add a ticket with known URL
	existingURL := "https://dup.example.com"
	ticketStorage.AddTicket("Existing", existingURL)

	// Test duplicate URL detection
	if !ticketStorage.HasTicketWithURL(existingURL) {
		t.Fatalf("expected ticket storage to detect duplicate URL")
	}
	
	if ticketStorage.HasTicketWithURL("https://nonexistent.com") {
		t.Fatalf("expected ticket storage to not find non-existent URL")
	}
}