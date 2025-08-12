package ui

import tea "github.com/charmbracelet/bubbletea"

// HandleBackups handles backup list navigation
func (m Model) HandleBackups(msg tea.KeyMsg) (Model, tea.Cmd) {
	newModel := m

	switch msg.String() {
	case "ctrl+c", "q":
		newModel.SetViewMode(ViewList)
		newModel.selectedBackupIndex = -1
		return newModel, nil
	case "up", "k":
		if len(newModel.backups) > 0 {
			if newModel.selectedBackupIndex > 0 {
				newModel.selectedBackupIndex--
			} else {
				newModel.selectedBackupIndex = len(newModel.backups) - 1
			}
		}
		return newModel, nil
	case "down", "j":
		if len(newModel.backups) > 0 {
			if newModel.selectedBackupIndex < len(newModel.backups)-1 {
				newModel.selectedBackupIndex++
			} else {
				newModel.selectedBackupIndex = 0
			}
		}
		return newModel, nil
	case "enter":
		if len(newModel.backups) > 0 && newModel.selectedBackupIndex >= 0 && newModel.selectedBackupIndex < len(newModel.backups) {
			newModel.backupToRestore = newModel.backups[newModel.selectedBackupIndex]
			newModel.SetViewMode(ViewConfirmRestore)
		}
		return newModel, nil
	case "esc":
		newModel.SetViewMode(ViewList)
		newModel.selectedBackupIndex = -1
		return newModel, nil
	}
	return newModel, nil
}