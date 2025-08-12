package ui

import (
	"gotickets/internal/storage"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode represents the current view state of the application
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewAddURL
	ViewAddTitle
	ViewConfirmDelete
	ViewImport
	ViewImportResult
	ViewBackups
	ViewConfirmRestore
)

// Model represents the main application state
type Model struct {
	storage             *storage.TicketStorage
	viewMode            ViewMode
	list                list.Model
	textInput           textinput.Model
	searchMode          bool
	searchQuery         string
	tempURL             string
	ticketToDelete      int
	importResult        *storage.ImportResult
	backups             []string
	backupToRestore     string
	selectedBackupIndex int
	urlError            string
}

// NewModel creates and initializes a new application model
func NewModel() Model {
	ticketStorage, _ := storage.LoadTicketsWithFS(&storage.RealFileSystem{})

	// Convert tickets to list items
	items := make([]list.Item, len(ticketStorage.Tickets))
	for i, ticket := range ticketStorage.Tickets {
		items[i] = ticket
	}

	// Create list with custom delegate
	listComponent := createList(items, len(ticketStorage.Tickets))

	// Create text input component
	textInputComponent := createTextInput()

	return Model{
		storage:             ticketStorage,
		viewMode:            ViewList,
		list:                listComponent,
		textInput:           textInputComponent,
		ticketToDelete:      -1,
		backups:             []string{},
		backupToRestore:     "",
		selectedBackupIndex: -1,
	}
}

// GetViewMode returns the current view mode
func (m Model) GetViewMode() ViewMode {
	return m.viewMode
}

// SetViewMode sets the view mode
func (m *Model) SetViewMode(mode ViewMode) {
	m.viewMode = mode
}

// GetStorage returns the ticket storage
func (m Model) GetStorage() *storage.TicketStorage {
	return m.storage
}

// SetStorage sets the ticket storage
func (m *Model) SetStorage(storage *storage.TicketStorage) {
	m.storage = storage
}

// IsSearchMode returns whether the model is in search mode
func (m Model) IsSearchMode() bool {
	return m.searchMode
}

// SetListSize sets the size of the list component
func (m *Model) SetListSize(width, height int) {
	m.list.SetWidth(width)
	m.list.SetHeight(height)
}

// UpdateList updates the list component with a message
func (m *Model) UpdateList(msg tea.Msg) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	_ = cmd // Ignore command for now
}

// Init implements tea.Model interface
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model interface - this will be overridden in main
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}
