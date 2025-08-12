package ui

import (
	"strings"

	"gotickets/internal/storage"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// HandleListView handles input for the main list view
func (m Model) HandleListView(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "enter":
		return m.handleEnterInList()
	case "a":
		return m.handleAddTicket()
	case "/":
		return m.handleSearch()
	case "r":
		m.RefreshList()
		return m, nil
	case "d":
		return m.handleDeleteTicket()
	case "o":
		return m.handleOpenTicket()
	case "i":
		return m.handleImport()
	case "b":
		return m.handleBackups()
	}

	// Let the list handle other keys (navigation, filtering, etc.)
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleEnterInList() (Model, tea.Cmd) {
	if selectedItem := m.list.SelectedItem(); selectedItem != nil {
		if ticket, ok := selectedItem.(storage.Ticket); ok && ticket.URL != "" {
			// Copy URL to clipboard
			clipboard.WriteAll(ticket.URL)
		}
	}
	return m, nil
}

func (m Model) handleAddTicket() (Model, tea.Cmd) {
	newModel := m
	newModel.SetViewMode(ViewAddURL)
	newModel.SetupTextInputForURL()
	return newModel, nil
}

func (m Model) handleSearch() (Model, tea.Cmd) {
	newModel := m
	newModel.SetupTextInputForSearch()
	return newModel, nil
}

func (m Model) handleDeleteTicket() (Model, tea.Cmd) {
	if selectedItem := m.list.SelectedItem(); selectedItem != nil {
		if ticket, ok := selectedItem.(storage.Ticket); ok {
			newModel := m
			newModel.ticketToDelete = ticket.ID
			newModel.SetViewMode(ViewConfirmDelete)
			return newModel, nil
		}
	}
	return m, nil
}

func (m Model) handleOpenTicket() (Model, tea.Cmd) {
	if selectedItem := m.list.SelectedItem(); selectedItem != nil {
		if ticket, ok := selectedItem.(storage.Ticket); ok && ticket.URL != "" {
			go openBrowser(ticket.URL)()
		}
	}
	return m, nil
}

func (m Model) handleImport() (Model, tea.Cmd) {
	newModel := m
	newModel.SetViewMode(ViewImport)
	newModel.SetupTextInputForImport()
	return newModel, nil
}

func (m Model) handleBackups() (Model, tea.Cmd) {
	backups, err := storage.ListBackupsUsing(&storage.RealFileSystem{})
	if err != nil {
		return m, nil
	}

	newModel := m
	newModel.backups = backups
	if len(backups) > 0 {
		newModel.selectedBackupIndex = 0
	} else {
		newModel.selectedBackupIndex = -1
	}
	newModel.SetViewMode(ViewBackups)
	return newModel, nil
}

// HandleAddURL handles input for URL entry
func (m Model) HandleAddURL(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel := m

	switch msg.String() {
	case "ctrl+c":
		return newModel, tea.Quit
	case "esc":
		newModel.SetViewMode(ViewList)
		newModel.ClearTextInput()
		newModel.tempURL = ""
		newModel.urlError = ""
		return newModel, nil
	case "enter":
		return m.handleURLSubmit()
	}

	// Clear error when user starts typing
	if newModel.urlError != "" {
		newModel.urlError = ""
	}

	// Let textinput handle the input
	newModel.textInput, cmd = newModel.textInput.Update(msg)
	return newModel, cmd
}

func (m Model) handleURLSubmit() (Model, tea.Cmd) {
	value := strings.TrimSpace(m.textInput.Value())
	if value == "" {
		return m, nil
	}

	// Check for duplicate URL
	if m.storage.HasTicketWithURL(value) {
		newModel := m
		newModel.urlError = "Тикет с такой ссылкой уже существует!"
		return newModel, nil
	}

	newModel := m
	newModel.tempURL = value
	newModel.SetViewMode(ViewAddTitle)
	newModel.SetupTextInputForTitle()
	return newModel, nil
}

// HandleAddTitle handles input for title entry
func (m Model) HandleAddTitle(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel := m

	switch msg.String() {
	case "ctrl+c":
		return newModel, tea.Quit
	case "esc":
		newModel.SetViewMode(ViewList)
		newModel.ClearTextInput()
		newModel.tempURL = ""
		return newModel, nil
	case "enter":
		return m.handleTitleSubmit()
	}

	// Let textinput handle the input
	newModel.textInput, cmd = newModel.textInput.Update(msg)
	return newModel, cmd
}

func (m Model) handleTitleSubmit() (Model, tea.Cmd) {
	value := strings.TrimSpace(m.textInput.Value())
	if value == "" {
		return m, nil
	}

	newModel := m
	newModel.storage.AddTicket(value, newModel.tempURL)
	newModel.storage.Save()
	newModel.RefreshList()
	newModel.SetViewMode(ViewList)
	newModel.ClearTextInput()
	newModel.tempURL = ""

	// Move to the last item (newly added ticket)
	if len(newModel.storage.Tickets) > 0 {
		newModel.list.Select(len(newModel.storage.Tickets) - 1)
	}
	return newModel, nil
}