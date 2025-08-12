package ui

import (
	"gotickets/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

// HandleConfirmDelete handles delete confirmation
func (m Model) HandleConfirmDelete(msg tea.KeyMsg) (Model, tea.Cmd) {
	newModel := m

	switch msg.String() {
	case "ctrl+c":
		return newModel, tea.Quit
	case "esc", "n":
		newModel.SetViewMode(ViewList)
		newModel.ticketToDelete = -1
	case "y", "enter":
		if newModel.ticketToDelete != -1 {
			if newModel.storage.DeleteTicket(newModel.ticketToDelete) {
				newModel.storage.Save()
				newModel.RefreshList()
			}
			newModel.SetViewMode(ViewList)
			newModel.ticketToDelete = -1
		}
	}
	return newModel, nil
}

// HandleConfirmRestore handles backup restore confirmation
func (m Model) HandleConfirmRestore(msg tea.KeyMsg) (Model, tea.Cmd) {
	newModel := m

	switch msg.String() {
	case "ctrl+c", "q":
		newModel.SetViewMode(ViewList)
		newModel.selectedBackupIndex = -1
		return newModel, nil
	case "y", "Y":
		// Confirm restore
		err := storage.RestoreFromBackupUsing(&storage.RealFileSystem{}, newModel.backupToRestore)
		if err != nil {
			return newModel, nil
		}
		// Reload tickets after restore
		storage, _ := storage.LoadTicketsWithFS(&storage.RealFileSystem{})
		newModel.SetStorage(storage)
		newModel.RefreshList()
		newModel.SetViewMode(ViewList)
		newModel.selectedBackupIndex = -1
		return newModel, nil
	case "n", "N", "esc":
		newModel.SetViewMode(ViewList)
		newModel.selectedBackupIndex = -1
		return newModel, nil
	}
	return newModel, nil
}