package main

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	
	if model.storage == nil {
		t.Error("Expected storage to be initialized")
	}
	if model.viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList, got %v", model.viewMode)
	}
	if model.textInput.Value() != "" {
		t.Errorf("Expected textInput to be empty, got '%s'", model.textInput.Value())
	}
	if model.tempURL != "" {
		t.Errorf("Expected tempURL to be empty, got '%s'", model.tempURL)
	}
	if model.ticketToDelete != -1 {
		t.Errorf("Expected ticketToDelete to be -1, got %d", model.ticketToDelete)
	}
	if model.list.Index() != 0 {
		t.Errorf("Expected list cursor to be 0, got %d", model.list.Index())
	}
}

func TestModel_Init(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	cmd := model.Init()
	
	// Init should return textinput.Blink command for cursor blinking
	if cmd == nil {
		t.Error("Expected Init() to return textinput.Blink command")
	}
}

func TestModel_UpdateList_Navigation(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create model with some test tickets
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	model.storage.AddTicket("Ticket 1", "https://example.com/1")
	model.storage.AddTicket("Ticket 2", "https://example.com/2")
	model.storage.AddTicket("Ticket 3", "https://example.com/3")
	model.refreshList()
	
	// Test that list has been updated
	if len(model.list.Items()) != 3 {
		t.Errorf("Expected 3 items in list, got %d", len(model.list.Items()))
	}
	
	// Test navigation is handled by list component
	updatedModel, _ := model.updateList(tea.KeyMsg{Type: tea.KeyUp})
	// The list component handles navigation internally
	if updatedModel == nil {
		t.Error("Expected model to be returned")
	}
}

func TestModel_UpdateList_ModeChanges(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	
	// Test add mode
	updatedModel, _ := model.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if updatedModel.(Model).viewMode != ViewAddURL {
		t.Errorf("Expected viewMode to be ViewAddURL after 'a', got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after 'a', got '%s'", updatedModel.(Model).textInput.Value())
	}
	
	// Test search mode
	updatedModel, _ = model.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if updatedModel.(Model).viewMode != ViewSearch {
		t.Errorf("Expected viewMode to be ViewSearch after 's', got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after 's', got '%s'", updatedModel.(Model).textInput.Value())
	}
	if !updatedModel.(Model).searchMode {
		t.Error("Expected searchMode to be true after 's'")
	}
	
	// Test refresh
	model.storage.AddTicket("Real Ticket", "https://example.com")
	updatedModel, _ = model.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if len(updatedModel.(Model).list.Items()) != len(model.storage.Tickets) {
		t.Error("Expected list to be refreshed after 'r'")
	}
	
	// Test delete mode with tickets
	model.storage.AddTicket("Test Ticket", "https://example.com")
	model.refreshList()
	updatedModel, _ = model.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if updatedModel.(Model).viewMode != ViewConfirmDelete {
		t.Errorf("Expected viewMode to be ViewConfirmDelete after 'd', got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).ticketToDelete == -1 {
		t.Error("Expected ticketToDelete to be set after 'd'")
	}
	
	// Test import mode
	updatedModel, _ = model.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if updatedModel.(Model).viewMode != ViewImport {
		t.Errorf("Expected viewMode to be ViewImport after 'i', got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after 'i', got '%s'", updatedModel.(Model).textInput.Value())
	}
}

func TestModel_UpdateAddURL(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	model.viewMode = ViewAddURL
	model.textInput.SetValue("https://example.com")
	
	// Test escape
	updatedModel, _ := model.updateAddURL(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after escape, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after escape, got '%s'", updatedModel.(Model).textInput.Value())
	}
	if updatedModel.(Model).tempURL != "" {
		t.Errorf("Expected tempURL to be cleared after escape, got '%s'", updatedModel.(Model).tempURL)
	}
	
	// Test that textinput component is working (we don't test internal key handling)
	model.textInput.SetValue("Test URL")
	if model.textInput.Value() != "Test URL" {
		t.Errorf("Expected textInput to store value correctly, got '%s'", model.textInput.Value())
	}
	
	// Test enter with valid input
	model.textInput.SetValue("https://example.com")
	updatedModel, _ = model.updateAddURL(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel := updatedModel.(Model)
	
	if finalModel.viewMode != ViewAddTitle {
		t.Errorf("Expected viewMode to be ViewAddTitle after enter, got %v", finalModel.viewMode)
	}
	if finalModel.textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after enter, got '%s'", finalModel.textInput.Value())
	}
	if finalModel.tempURL != "https://example.com" {
		t.Errorf("Expected tempURL to be set after enter, got '%s'", finalModel.tempURL)
	}
	
	// Test enter with empty input (should not proceed)
	model.textInput.SetValue("   ")
	model.tempURL = ""
	updatedModel, _ = model.updateAddURL(tea.KeyMsg{Type: tea.KeyEnter})
	if updatedModel.(Model).viewMode != ViewAddURL {
		t.Error("Expected to stay in ViewAddURL for empty input")
	}
}

func TestModel_UpdateAddTitle(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	model.viewMode = ViewAddTitle
	model.tempURL = "https://example.com"
	model.textInput.SetValue("Test Ticket")
	
	// Test escape
	updatedModel, _ := model.updateAddTitle(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after escape, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after escape, got '%s'", updatedModel.(Model).textInput.Value())
	}
	if updatedModel.(Model).tempURL != "" {
		t.Errorf("Expected tempURL to be cleared after escape, got '%s'", updatedModel.(Model).tempURL)
	}
	
	// Test enter with valid input
	initialTicketCount := len(model.storage.Tickets)
	model.textInput.SetValue("New Ticket")
	model.tempURL = "https://example.com"
	updatedModel, _ = model.updateAddTitle(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel := updatedModel.(Model)
	
	if finalModel.viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after enter, got %v", finalModel.viewMode)
	}
	if finalModel.textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after enter, got '%s'", finalModel.textInput.Value())
	}
	if finalModel.tempURL != "" {
		t.Errorf("Expected tempURL to be cleared after enter, got '%s'", finalModel.tempURL)
	}
	if len(finalModel.storage.Tickets) != initialTicketCount+1 {
		t.Error("Expected ticket to be added after enter")
	}
	// With list component, check that last item is selected
	if len(finalModel.storage.Tickets) > 0 && finalModel.list.Index() != len(finalModel.storage.Tickets)-1 {
		t.Error("Expected list selection to move to last ticket after adding")
	}
	
	// Verify ticket was created with correct URL and title
	if len(finalModel.storage.Tickets) > 0 {
		lastTicket := finalModel.storage.Tickets[len(finalModel.storage.Tickets)-1]
		if lastTicket.Title != "New Ticket" {
			t.Errorf("Expected ticket title to be 'New Ticket', got '%s'", lastTicket.Title)
		}
		if lastTicket.URL != "https://example.com" {
			t.Errorf("Expected ticket URL to be 'https://example.com', got '%s'", lastTicket.URL)
		}
	}
}

func TestModel_UpdateSearch(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create a properly initialized model
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	model.viewMode = ViewSearch
	model.storage.AddTicket("Unique Test Ticket", "https://example.com")
	model.storage.AddTicket("Another Ticket", "https://another.com")
	model.refreshList()
	model.textInput.SetValue("Unique")
	
	// Test escape
	updatedModel, _ := model.updateSearch(tea.KeyMsg{Type: tea.KeyEsc})
	finalModel := updatedModel.(Model)
	if finalModel.viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after escape, got %v", finalModel.viewMode)
	}
	if finalModel.textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after escape, got '%s'", finalModel.textInput.Value())
	}
	if len(finalModel.list.Items()) != len(finalModel.storage.Tickets) {
		t.Error("Expected list to be reset to all tickets after escape")
	}
	
	// Test enter
	model.textInput.SetValue("Unique")
	updatedModel, _ = model.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel = updatedModel.(Model)
	if finalModel.viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after enter, got %v", finalModel.viewMode)
	}
	if finalModel.textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after enter, got '%s'", finalModel.textInput.Value())
	}
	if len(finalModel.list.Items()) != 1 {
		t.Errorf("Expected 1 filtered ticket after search, got %d", len(finalModel.list.Items()))
	}
	
	// Test that textinput component is working
	model.textInput.SetValue("Search query")
	if model.textInput.Value() != "Search query" {
		t.Errorf("Expected textInput to store search value correctly, got '%s'", model.textInput.Value())
	}
}

func TestModel_UpdateConfirmDelete(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Create a properly initialized model
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	model.viewMode = ViewConfirmDelete
	model.storage.AddTicket("Test Ticket", "https://example.com")
	model.refreshList()
	model.ticketToDelete = 1
	
	// Test escape (cancel)
	updatedModel, _ := model.updateConfirmDelete(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after escape, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).ticketToDelete != -1 {
		t.Errorf("Expected ticketToDelete to be reset after escape, got %d", updatedModel.(Model).ticketToDelete)
	}
	
	// Test 'n' (cancel)
	model.viewMode = ViewConfirmDelete
	model.ticketToDelete = 1
	updatedModel, _ = model.updateConfirmDelete(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after 'n', got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).ticketToDelete != -1 {
		t.Errorf("Expected ticketToDelete to be reset after 'n', got %d", updatedModel.(Model).ticketToDelete)
	}
	
	// Test 'y' (confirm delete)
	initialCount := len(model.storage.Tickets)
	model.viewMode = ViewConfirmDelete
	model.ticketToDelete = 1
	updatedModel, _ = model.updateConfirmDelete(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	finalModel := updatedModel.(Model)
	
	if finalModel.viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after 'y', got %v", finalModel.viewMode)
	}
	if finalModel.ticketToDelete != -1 {
		t.Errorf("Expected ticketToDelete to be reset after 'y', got %d", finalModel.ticketToDelete)
	}
	if len(finalModel.storage.Tickets) != initialCount-1 {
		t.Errorf("Expected ticket count to decrease by 1, got %d", len(finalModel.storage.Tickets))
	}
	
	// Test enter (confirm delete)
	model.storage.AddTicket("Another Test", "https://test.com")
	model.viewMode = ViewConfirmDelete
	model.ticketToDelete = model.storage.Tickets[0].ID
	updatedModel, _ = model.updateConfirmDelete(tea.KeyMsg{Type: tea.KeyEnter})
	if updatedModel.(Model).viewMode != ViewList {
		t.Error("Expected to return to ViewList after enter confirm")
	}
}

func TestViewModes(t *testing.T) {
	tests := []struct {
		mode ViewMode
		name string
	}{
		{ViewList, "ViewList"},
		{ViewAddURL, "ViewAddURL"},
		{ViewAddTitle, "ViewAddTitle"},
		{ViewSearch, "ViewSearch"},
		{ViewConfirmDelete, "ViewConfirmDelete"},
		{ViewImport, "ViewImport"},
		{ViewImportResult, "ViewImportResult"},
	}
	
	for i, test := range tests {
		if int(test.mode) != i {
			t.Errorf("Expected %s to have value %d, got %d", test.name, i, int(test.mode))
		}
	}
}

func TestModel_View(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	model.storage.AddTicket("Test Ticket", "https://example.com")
	model.refreshList()
	model.ticketToDelete = 1
	
	// Test that View() returns a string for each mode
	modes := []ViewMode{ViewList, ViewAddURL, ViewAddTitle, ViewSearch, ViewConfirmDelete, ViewImport, ViewImportResult}
	
	for _, mode := range modes {
		model.viewMode = mode
		if mode == ViewImportResult {
			model.importResult = &ImportResult{Added: 1, Duplicates: 0, Errors: 0}
		}
		view := model.View()
		if view == "" {
			t.Errorf("Expected non-empty view for mode %v", mode)
		}
	}
}

func TestModel_UpdateImport(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	model.viewMode = ViewImport
	model.textInput.SetValue("/path/to/test.txt")
	
	// Test escape
	updatedModel, _ := model.updateImport(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after escape, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after escape, got '%s'", updatedModel.(Model).textInput.Value())
	}
	
	// Test that textinput component is working
	model.textInput.SetValue("Test File Path")
	if model.textInput.Value() != "Test File Path" {
		t.Errorf("Expected textInput to store path correctly, got '%s'", model.textInput.Value())
	}
	
	// Test enter with non-existent file (should show error result)
	model.textInput.SetValue("/non/existent/file.txt")
	updatedModel, _ = model.updateImport(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel := updatedModel.(Model)
	
	if finalModel.viewMode != ViewImportResult {
		t.Errorf("Expected viewMode to be ViewImportResult after enter, got %v", finalModel.viewMode)
	}
	if finalModel.textInput.Value() != "" {
		t.Errorf("Expected textInput to be cleared after enter, got '%s'", finalModel.textInput.Value())
	}
	if finalModel.importResult == nil {
		t.Error("Expected importResult to be set after failed import")
	} else if finalModel.importResult.Errors != 1 {
		t.Errorf("Expected 1 error for non-existent file, got %d", finalModel.importResult.Errors)
	}
	
	// Test enter with empty input (should not proceed)
	model.textInput.SetValue("   ")
	model.viewMode = ViewImport
	updatedModel, _ = model.updateImport(tea.KeyMsg{Type: tea.KeyEnter})
	if updatedModel.(Model).viewMode != ViewImport {
		t.Error("Expected to stay in ViewImport for empty input")
	}
}

func TestModel_UpdateImportResult(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	model.viewMode = ViewImportResult
	model.importResult = &ImportResult{Added: 5, Duplicates: 2, Errors: 1}
	
	// Test escape
	updatedModel, _ := model.updateImportResult(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after escape, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).importResult != nil {
		t.Error("Expected importResult to be cleared after escape")
	}
	
	// Test enter
	model.importResult = &ImportResult{Added: 3, Duplicates: 1, Errors: 0}
	updatedModel, _ = model.updateImportResult(tea.KeyMsg{Type: tea.KeyEnter})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after enter, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).importResult != nil {
		t.Error("Expected importResult to be cleared after enter")
	}
	
	// Test space
	model.importResult = &ImportResult{Added: 1, Duplicates: 0, Errors: 0}
	updatedModel, _ = model.updateImportResult(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if updatedModel.(Model).viewMode != ViewList {
		t.Errorf("Expected viewMode to be ViewList after space, got %v", updatedModel.(Model).viewMode)
	}
	if updatedModel.(Model).importResult != nil {
		t.Error("Expected importResult to be cleared after space")
	}
}

func TestModel_ListComponent(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	// Test that the list component works correctly
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	
	// Add some tickets
	for i := 0; i < 10; i++ {
		model.storage.AddTicket(fmt.Sprintf("Ticket %d", i+1), fmt.Sprintf("https://example.com/%d", i+1))
	}
	model.refreshList()
	
	// Test that list has correct number of items
	if len(model.list.Items()) != 10 {
		t.Errorf("Expected 10 items in list, got %d", len(model.list.Items()))
	}
	
	// Test that first item is correct
	if model.list.Items()[0].(Ticket).Title != "Ticket 1" {
		t.Errorf("Expected first item to be 'Ticket 1', got '%s'", model.list.Items()[0].(Ticket).Title)
	}
}

func TestModel_ListNavigation(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	
	// Add some tickets
	for i := 0; i < 10; i++ {
		model.storage.AddTicket(fmt.Sprintf("Ticket %d", i+1), fmt.Sprintf("https://example.com/%d", i+1))
	}
	model.refreshList()
	
	// Test that navigation doesn't crash - list component handles this internally
	updatedModel, _ := model.updateList(tea.KeyMsg{Type: tea.KeyUp})
	if updatedModel == nil {
		t.Error("Expected model to be returned")
	}
	
	updatedModel, _ = model.updateList(tea.KeyMsg{Type: tea.KeyDown})
	if updatedModel == nil {
		t.Error("Expected model to be returned")
	}
}

func TestModel_ListWithEmptyList(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.refreshList()
	
	// Test with empty list
	if len(model.list.Items()) != 0 {
		t.Errorf("Expected 0 items for empty list, got %d", len(model.list.Items()))
	}
	
	// Test navigation with empty list (should not crash)
	updatedModel, _ := model.updateList(tea.KeyMsg{Type: tea.KeyUp})
	if updatedModel == nil {
		t.Error("Expected model to be returned even with empty list")
	}
}

func TestModel_UpdateList_EnterCopiesURL(t *testing.T) {
	// Set up mock file system
	tempDir := t.TempDir()
	mockFS := NewMockFileSystem(tempDir)
	originalFS := fileSystem
	fileSystem = mockFS
	defer func() { fileSystem = originalFS }()
	
	model := NewModel()
	// Clear existing tickets first
	model.storage.Tickets = []Ticket{}
	model.storage.NextID = 1
	
	// Add a test ticket
	model.storage.AddTicket("Test Ticket", "https://github.com/test/repo/issues/123")
	model.refreshList()
	
	// Test Enter key to copy URL
	updatedModel, _ := model.updateList(tea.KeyMsg{Type: tea.KeyEnter})
	if updatedModel == nil {
		t.Error("Expected model to be returned after Enter")
	}
	
	// Note: We can't easily test clipboard content in automated tests
	// but we can verify the key handler doesn't crash
}