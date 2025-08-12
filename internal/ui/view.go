package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the current view based on the view mode
func (m Model) View() string {
	switch m.viewMode {
	case ViewList:
		return m.renderListView()
	case ViewAddURL:
		return m.renderAddURLView()
	case ViewAddTitle:
		return m.renderAddTitleView()
	case ViewConfirmDelete:
		return m.renderConfirmDeleteView()
	case ViewImport:
		return m.renderImportView()
	case ViewImportResult:
		return m.renderImportResultView()
	case ViewBackups:
		return m.renderBackupsView()
	case ViewConfirmRestore:
		return m.renderConfirmRestoreView()
	default:
		return "Unknown view mode"
	}
}

func (m Model) renderListView() string {
	if m.searchMode {
		// Show list with search input at bottom
		var s strings.Builder
		s.WriteString(m.list.View())
		s.WriteString("\n")
		s.WriteString(m.getInputStyle().Render("Поиск: " + m.textInput.View()))
		s.WriteString("\n")
		s.WriteString(m.formatKeyHelp("Enter", "применить поиск", "Esc", "отмена"))
		return s.String()
	}
	return m.list.View()
}

func (m Model) renderAddURLView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Добавить новый тикет - Ссылка"))
	s.WriteString("\n\n")
	s.WriteString("Введите ссылку:\n")
	s.WriteString(m.getInputStyle().Render(m.textInput.View()))
	s.WriteString("\n")

	// Show error message if there is one
	if m.urlError != "" {
		s.WriteString(m.getErrorStyle().Render("❌ " + m.urlError))
		s.WriteString("\n")
	}

	s.WriteString(m.formatKeyHelp("Enter", "продолжить к названию", "Esc", "отмена"))
	return s.String()
}

func (m Model) renderAddTitleView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Добавить новый тикет - Название"))
	s.WriteString("\n\n")
	s.WriteString(fmt.Sprintf("Ссылка: %s\n", m.tempURL))
	s.WriteString("Введите название тикета:\n")
	s.WriteString(m.getInputStyle().Render(m.textInput.View()))
	s.WriteString("\n")
	s.WriteString(m.formatKeyHelp("Enter", "сохранить тикет", "Esc", "отмена"))
	return s.String()
}

func (m Model) renderConfirmDeleteView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Подтверждение удаления"))
	s.WriteString("\n\n")

	// Find the ticket to delete
	var ticketTitle string
	for _, ticket := range m.storage.Tickets {
		if ticket.ID == m.ticketToDelete {
			ticketTitle = ticket.Title
			break
		}
	}

	s.WriteString(fmt.Sprintf("Вы уверены, что хотите удалить тикет?\n"))
	s.WriteString(fmt.Sprintf("Тикет: #%d - %s\n", m.ticketToDelete, ticketTitle))
	s.WriteString(m.formatKeyHelp("y/Enter", "да, удалить", "n/Esc", "нет, отменить"))
	return s.String()
}

func (m Model) renderImportView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Импорт тикетов"))
	s.WriteString("\n\n")
	s.WriteString("Введите путь к .txt файлу:\n")
	s.WriteString("Формат файла: каждая строка содержит 'URL - Название'\n\n")
	s.WriteString(m.getInputStyle().Render(m.textInput.View()))
	s.WriteString("\n")
	s.WriteString(m.formatKeyHelp("Enter", "начать импорт", "Esc", "отмена"))
	return s.String()
}

func (m Model) renderImportResultView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Результат импорта"))
	s.WriteString("\n\n")

	if m.importResult != nil {
		s.WriteString(fmt.Sprintf("✅ Добавлено тикетов: %d\n", m.importResult.Added))
		s.WriteString(fmt.Sprintf("🔄 Дубликатов пропущено: %d\n", m.importResult.Duplicates))
		s.WriteString(fmt.Sprintf("❌ Ошибок формата: %d\n", m.importResult.Errors))

		if len(m.importResult.ErrorLines) > 0 {
			s.WriteString("\nОшибки:\n")
			for _, errLine := range m.importResult.ErrorLines {
				s.WriteString(fmt.Sprintf("  • %s\n", errLine))
			}
		}
	}

	s.WriteString("\n")
	s.WriteString(m.formatKeyHelp("Enter/Esc/Пробел", "вернуться к списку"))
	return s.String()
}

func (m Model) renderBackupsView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Резервные копии"))
	s.WriteString("\n\n")

	if len(m.backups) == 0 {
		s.WriteString("Резервные копии не найдены.\n")
	} else {
		s.WriteString(fmt.Sprintf("Найдено резервных копий: %d\n\n", len(m.backups)))
		for i, backup := range m.backups {
			if i == m.selectedBackupIndex {
				// Highlight selected backup
				s.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("15")).
					Background(lipgloss.Color("12")).
					Padding(0, 1).
					Render(fmt.Sprintf("> %s", backup)))
			} else {
				s.WriteString(fmt.Sprintf("  %s", backup))
			}
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(m.formatKeyHelp("↑/↓", "навигация", "Enter", "выбрать", "Esc", "отмена"))
	return s.String()
}

func (m Model) renderConfirmRestoreView() string {
	var s strings.Builder
	s.WriteString(m.getHeaderStyle().Render("Подтверждение восстановления"))
	s.WriteString("\n\n")
	s.WriteString(fmt.Sprintf("Вы уверены, что хотите восстановить из резервной копии?\n"))
	s.WriteString(fmt.Sprintf("Резервная копия: %s\n\n", m.backupToRestore))
	s.WriteString("⚠️  ВНИМАНИЕ: Это заменит все текущие тикеты!\n")
	s.WriteString(m.formatKeyHelp("y/Enter", "да, восстановить", "n/Esc", "нет, отменить"))
	return s.String()
}

// Style helpers
func (m Model) getHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Padding(1, 2)
}

func (m Model) getInputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)
}

func (m Model) getErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)
}

func (m Model) getKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)
}

func (m Model) getActionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))
}

func (m Model) getHelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginTop(1)
}

// formatKeyHelp formats key help pairs
func (m Model) formatKeyHelp(pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	var helpParts []string
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		action := pairs[i+1]
		helpParts = append(helpParts, m.getKeyStyle().Render(key)+" - "+m.getActionStyle().Render(action))
	}
	return m.getHelpStyle().Render(strings.Join(helpParts, " • "))
}