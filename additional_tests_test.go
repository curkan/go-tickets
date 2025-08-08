package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTicket_FilterValue_And_GetDescription(t *testing.T) {
	ticket := Ticket{ID: 1, Title: "Test", URL: "https://example.com/x"}
	if got, want := ticket.FilterValue(), "Test https://example.com/x"; got != want {
		t.Errorf("FilterValue() = %q, want %q", got, want)
	}
	if got, want := ticket.GetDescription(), ticket.URL; got != want {
		t.Errorf("GetDescription() = %q, want %q", got, want)
	}
}

func TestCreateBackup_NoExistingTicketsFile(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	// No tickets.json exists → should be no-op without error
	if err := createBackup(); err != nil {
		t.Fatalf("createBackup() returned error for non-existent file: %v", err)
	}
}

func TestCreateBackup_ReadFileError(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	dataDir := filepath.Join(tempDir, ".gotickets")
	ticketsPath := filepath.Join(dataDir, "tickets.json")

	// Seed a tickets.json file
	if err := mockFS.WriteFile(ticketsPath, []byte(`{"tickets":[],"next_id":1}`), 0644); err != nil {
		t.Fatalf("failed to seed tickets.json: %v", err)
	}
	// Force ReadFile error
	mockFS.errors["ReadFile"] = assertErr("read failure")

	if err := createBackup(); err == nil || !strings.Contains(err.Error(), "failed to read tickets file for backup") {
		t.Fatalf("expected read error from createBackup, got: %v", err)
	}
}

func TestListBackups_ErrorFromReadDir(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	mockFS.errors["ReadDir"] = assertErr("boom")
	if _, err := listBackups(); err == nil {
		t.Fatal("expected error from listBackups when ReadDir fails")
	}
}

func TestLoadTickets_UserHomeDirError(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	mockFS.errors["UserHomeDir"] = assertErr("cannot resolve home")
	storage, err := LoadTickets()
	if err != nil {
		t.Fatalf("LoadTickets should not return error on UserHomeDir failure: %v", err)
	}
	if storage == nil || storage.NextID != 1 || len(storage.Tickets) != 0 {
		t.Fatalf("unexpected storage returned: %+v", storage)
	}
}

func TestModel_UpdateAddURL_DuplicateShowsError(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	model := NewModel()
	// Ensure clean state
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	// Add a ticket with known URL
	existingURL := "https://dup.example.com"
	model.storage.AddTicket("Existing", existingURL)

	// Try to add the same URL
	model.viewMode = ViewAddURL
	model.textInput.SetValue(existingURL)
	updated, _ := model.updateAddURL(tea.KeyMsg{Type: tea.KeyEnter})
	res := updated.(Model)
	if res.viewMode != ViewAddURL {
		t.Fatalf("expected to stay in ViewAddURL, got %v", res.viewMode)
	}
	if res.urlError == "" || !strings.Contains(res.urlError, "уже существует") {
		t.Fatalf("expected duplicate URL error, got %q", res.urlError)
	}
}

func TestModel_UpdateImport_Success(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	// Prepare import file inside mock FS
	importPath := filepath.Join(tempDir, "import.txt")
	if err := mockFS.WriteFile(importPath, []byte("https://example.com/42 - The Answer\n"), 0644); err != nil {
		t.Fatalf("failed to write import file: %v", err)
	}

	model := NewModel()
	// Start from clean state
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	model.refreshList()

	model.viewMode = ViewImport
	model.textInput.SetValue(importPath)
	updated, _ := model.updateImport(tea.KeyMsg{Type: tea.KeyEnter})
	res := updated.(Model)

	if res.viewMode != ViewImportResult {
		t.Fatalf("expected ViewImportResult, got %v", res.viewMode)
	}
	if res.importResult == nil || res.importResult.Added != 1 || res.importResult.Errors != 0 {
		t.Fatalf("unexpected importResult: %+v", res.importResult)
	}
	if len(res.storage.Tickets) != 1 {
		t.Fatalf("expected 1 ticket after import, got %d", len(res.storage.Tickets))
	}
	// After refreshList, selection should move to last item (index 0 here)
	if res.list.Index() != len(res.storage.Tickets)-1 {
		t.Fatalf("expected list selection at last item, got %d", res.list.Index())
	}
}

func TestModel_Backups_Navigation_And_Restore(t *testing.T) {
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()

	// Seed two backup files
	dataDir := filepath.Join(tempDir, ".gotickets")
	backup1 := "tickets_backup_2024-01-01_00-00-00.json"
	backup2 := "tickets_backup_2024-01-02_00-00-00.json"
	// backup1 contains one ticket
	b1 := `{"tickets":[{"id":1,"title":"Restored","url":"https://ex.com","created_at":"2024-01-01T00:00:00Z"}],"next_id":2}`
	if err := mockFS.WriteFile(filepath.Join(dataDir, backup1), []byte(b1), 0644); err != nil {
		t.Fatalf("failed to seed backup1: %v", err)
	}
	if err := mockFS.WriteFile(filepath.Join(dataDir, backup2), []byte(`{"tickets":[],"next_id":1}`), 0644); err != nil {
		t.Fatalf("failed to seed backup2: %v", err)
	}

	model := NewModel()
	// Enter backups view via 'b'
	updated, _ := model.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	res := updated.(Model)
	if res.viewMode != ViewBackups {
		t.Fatalf("expected ViewBackups, got %v", res.viewMode)
	}
	if len(res.backups) != 2 || res.selectedBackupIndex != 0 {
		t.Fatalf("expected 2 backups and selection at 0, got len=%d idx=%d", len(res.backups), res.selectedBackupIndex)
	}

	// Navigate up (wrap to last)
	updated, _ = res.updateBackups(tea.KeyMsg{Type: tea.KeyUp})
	res = updated.(Model)
	if res.selectedBackupIndex != 1 {
		t.Fatalf("expected wrap to last index 1, got %d", res.selectedBackupIndex)
	}
	// Navigate down (wrap to first)
	updated, _ = res.updateBackups(tea.KeyMsg{Type: tea.KeyDown})
	res = updated.(Model)
	if res.selectedBackupIndex != 0 {
		t.Fatalf("expected wrap to first index 0, got %d", res.selectedBackupIndex)
	}

	// Select current backup (backup1) and confirm restore 'y'
	updated, _ = res.updateBackups(tea.KeyMsg{Type: tea.KeyEnter})
	res = updated.(Model)
	if res.viewMode != ViewConfirmRestore || res.backupToRestore == "" {
		t.Fatalf("expected ViewConfirmRestore with selected backup, got mode=%v name=%q", res.viewMode, res.backupToRestore)
	}
	// Confirm restore
	updated, _ = res.updateConfirmRestore(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	res = updated.(Model)
	if res.viewMode != ViewList {
		t.Fatalf("expected ViewList after restore, got %v", res.viewMode)
	}
	if len(res.storage.Tickets) != 1 || res.storage.Tickets[0].Title != "Restored" {
		t.Fatalf("restore did not load expected tickets, got %+v", res.storage.Tickets)
	}
}

// assertErr is a helper to produce deterministic errors via MockFileSystem.errors
type assertErr string

func (e assertErr) Error() string { return string(e) }
