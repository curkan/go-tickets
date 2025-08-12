package ui

import tea "github.com/charmbracelet/bubbletea"

// HandleSearch handles input during search mode
func (m Model) HandleSearch(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel := m

	switch msg.String() {
	case "ctrl+c":
		return newModel, tea.Quit
	case "esc":
		newModel.searchMode = false
		newModel.searchQuery = ""
		newModel.ClearTextInput()
		newModel.RefreshList() // Reset to show all tickets
		return newModel, nil
	case "enter":
		// Apply current search and exit search mode
		newModel.searchMode = false
		newModel.searchQuery = newModel.textInput.Value()
		newModel.textInput.Blur()
		newModel.FilterList(newModel.searchQuery)
		return newModel, nil
	case "up", "down":
		// Handle navigation within the list while in search mode
		var listCmd tea.Cmd
		newModel.list, listCmd = newModel.list.Update(msg)
		return newModel, listCmd
	}

	// Let textinput handle the input and update search in real time
	newModel.textInput, cmd = newModel.textInput.Update(msg)

	// Filter tickets in real time as user types
	query := newModel.textInput.Value()
	newModel.FilterList(query)

	return newModel, cmd
}