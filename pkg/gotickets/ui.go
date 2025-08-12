package gotickets

import (
	"gotickets/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

// Model wraps the internal UI model
type Model struct {
	ui.Model
}

// ViewMode type alias for backward compatibility  
type ViewMode = ui.ViewMode

// ViewMode constants for backward compatibility
const (
	ViewList          = ui.ViewList
	ViewAddURL        = ui.ViewAddURL
	ViewAddTitle      = ui.ViewAddTitle
	ViewConfirmDelete = ui.ViewConfirmDelete
	ViewImport        = ui.ViewImport
	ViewImportResult  = ui.ViewImportResult
	ViewBackups       = ui.ViewBackups
	ViewConfirmRestore = ui.ViewConfirmRestore
)

// NewModel creates a new UI model
func NewModel() Model {
	return Model{ui.NewModel()}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all input events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetListSize(msg.Width, msg.Height-10) // Leave room for title with border and status
		return m, nil
		
	case tea.KeyMsg:
		if m.IsSearchMode() {
			model, cmd := m.HandleSearch(msg)
			return Model{model}, cmd
		}
		
		switch m.GetViewMode() {
		case ViewList:
			model, cmd := m.HandleListView(msg)
			return Model{model}, cmd
		case ViewAddURL:
			model, cmd := m.HandleAddURL(msg)
			return Model{model}, cmd
		case ViewAddTitle:
			model, cmd := m.HandleAddTitle(msg)
			return Model{model}, cmd
		case ViewConfirmDelete:
			model, cmd := m.HandleConfirmDelete(msg)
			return Model{model}, cmd
		case ViewImport:
			model, cmd := m.HandleImport(msg)
			return Model{model}, cmd
		case ViewImportResult:
			model, cmd := m.HandleImportResult(msg)
			return Model{model}, cmd
		case ViewBackups:
			model, cmd := m.HandleBackups(msg)
			return Model{model}, cmd
		case ViewConfirmRestore:
			model, cmd := m.HandleConfirmRestore(msg)
			return Model{model}, cmd
		}
	}
	
	// Update the list component for other message types
	var cmd tea.Cmd
	m.UpdateList(msg)
	return m, cmd
}

// View renders the view
func (m Model) View() string {
	return m.Model.View()
}