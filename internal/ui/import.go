package ui

import (
	"strings"

	"gotickets/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

// HandleImport handles file import input
func (m Model) HandleImport(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel := m

	switch msg.String() {
	case "ctrl+c":
		return newModel, tea.Quit
	case "esc":
		newModel.SetViewMode(ViewList)
		newModel.ClearTextInput()
		return newModel, nil
	case "enter":
		return m.handleImportSubmit()
	}

	// Let textinput handle the input
	newModel.textInput, cmd = newModel.textInput.Update(msg)
	return newModel, cmd
}

func (m Model) handleImportSubmit() (Model, tea.Cmd) {
	filePath := strings.TrimSpace(m.textInput.Value())
	if filePath == "" {
		return m, nil
	}

	newModel := m
	result, err := newModel.storage.ImportFromFile(filePath)
	if err != nil {
		// Show error as result with 0 added tickets
		newModel.importResult = &storage.ImportResult{
			Added:      0,
			Duplicates: 0,
			Errors:     1,
			ErrorLines: []string{err.Error()},
		}
	} else {
		newModel.importResult = result
		// Save changes if tickets were added
		if result.Added > 0 {
			newModel.storage.Save()
			newModel.RefreshList()
			// Navigate to last item to show newly imported tickets
			if len(newModel.storage.Tickets) > 0 {
				newModel.list.Select(len(newModel.storage.Tickets) - 1)
			}
		}
	}

	newModel.SetViewMode(ViewImportResult)
	newModel.ClearTextInput()
	return newModel, nil
}

// HandleImportResult handles import result display
func (m Model) HandleImportResult(msg tea.KeyMsg) (Model, tea.Cmd) {
	newModel := m

	switch msg.String() {
	case "enter", " ":
		newModel.SetViewMode(ViewList)
		newModel.importResult = nil
		newModel.RefreshList()
		return newModel, nil
	case "esc":
		newModel.SetViewMode(ViewList)
		newModel.importResult = nil
		return newModel, nil
	}
	return newModel, nil
}