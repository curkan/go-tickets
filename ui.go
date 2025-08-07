package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

// Custom delegate for rendering tickets
type ticketDelegate struct{}

func (d ticketDelegate) Height() int                               { return 1 }
func (d ticketDelegate) Spacing() int                              { return 0 }
func (d ticketDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d ticketDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	ticket, ok := listItem.(Ticket)
	if !ok {
		return
	}

	// Use the new format with SCR # and extracted ticket number
	ticketNum := ticket.ExtractTicketNumber()
	str := fmt.Sprintf("SCR #%s - %s", ticketNum, ticket.Title)

	if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("12")).
			Padding(0, 1).
			Render("> " + str))
	} else {
		fmt.Fprint(w, lipgloss.NewStyle().PaddingLeft(4).Render(str))
	}
}

type Model struct {
	storage        *TicketStorage
	viewMode       ViewMode
	list           list.Model
	textInput      textinput.Model
	searchMode     bool
	searchQuery    string
	tempURL        string
	ticketToDelete int
	importResult   *ImportResult
	backups        []string
	backupToRestore string
	selectedBackupIndex int
}

func NewModel() Model {
	storage, _ := LoadTickets()
	
	// Convert tickets to list items
	items := make([]list.Item, len(storage.Tickets))
	for i, ticket := range storage.Tickets {
		items[i] = ticket
	}
	
	// Create list with custom delegate
	l := list.New(items, ticketDelegate{}, 80, 24)
	l.Title = fmt.Sprintf("%s\nВсего тикетов: %d", 
		lipgloss.NewStyle().Bold(true).Render("GoTickets - Ticket Manager"), 
		len(storage.Tickets))
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // We'll handle search manually
	l.Styles.Title = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Align(lipgloss.Center)
	
	// Set status bar message
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "copy url")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open")),
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import")),
			key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "backups")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		}
	}
	
	// Create text input component
	ti := textinput.New()
	ti.Placeholder = "Enter text..."
	ti.CharLimit = 500
	ti.Width = 60
	// Start without focus since we begin in ViewList mode
	
	return Model{
		storage:        storage,
		viewMode:       ViewList,
		list:           l,
		textInput:      ti,
		ticketToDelete: -1,
		backups:        []string{},
		backupToRestore: "",
		selectedBackupIndex: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 10) // Leave room for title with border and status
		return m, nil
		
	case tea.KeyMsg:
		if m.searchMode {
			return m.updateSearch(msg)
		}
		
		switch m.viewMode {
		case ViewList:
			return m.updateList(msg)
		case ViewAddURL:
			return m.updateAddURL(msg)
		case ViewAddTitle:
			return m.updateAddTitle(msg)
		case ViewConfirmDelete:
			return m.updateConfirmDelete(msg)
		case ViewImport:
			return m.updateImport(msg)
		case ViewImportResult:
			return m.updateImportResult(msg)
		case ViewBackups:
			return m.updateBackups(msg)
		case ViewConfirmRestore:
			return m.updateConfirmRestore(msg)
		}
	}
	
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "enter":
		if selectedItem := m.list.SelectedItem(); selectedItem != nil {
			if ticket, ok := selectedItem.(Ticket); ok && ticket.URL != "" {
				// Copy URL to clipboard
				err := clipboard.WriteAll(ticket.URL)
				if err == nil {
					// TODO: Show success message
					_ = err // Ignore error for now
				}
			}
		}
		return m, nil
	case "a":
		m.viewMode = ViewAddURL
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Enter URL..."
		m.textInput.Focus()
		m.tempURL = ""
		return m, nil
	case "/":
		m.searchMode = true
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Search tickets..."
		m.textInput.Focus()
		return m, nil
	case "r":
		m.refreshList()
		return m, nil
	case "d":
		if selectedItem := m.list.SelectedItem(); selectedItem != nil {
			if ticket, ok := selectedItem.(Ticket); ok {
				m.ticketToDelete = ticket.ID
				m.viewMode = ViewConfirmDelete
				return m, nil
			}
		}
	case "o":
		if selectedItem := m.list.SelectedItem(); selectedItem != nil {
			if ticket, ok := selectedItem.(Ticket); ok && ticket.URL != "" {
				go openBrowser(ticket.URL)()
			}
		}
		return m, nil
	case "i":
		m.viewMode = ViewImport
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Enter path to .txt file..."
		m.textInput.Focus()
		return m, nil
	case "b":
		backups, err := listBackups()
		if err != nil {
			// TODO: Show error message
			return m, nil
		}
		m.backups = backups
		if len(backups) > 0 {
			m.selectedBackupIndex = 0 // Initialize to first backup
		} else {
			m.selectedBackupIndex = -1 // No backups available
		}
		m.viewMode = ViewBackups
		return m, nil
	}
	
	// Let the list handle other keys (navigation, filtering, etc.)
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) refreshList() {
	items := make([]list.Item, len(m.storage.Tickets))
	for i, ticket := range m.storage.Tickets {
		items[i] = ticket
	}
	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("%s\nВсего тикетов: %d", 
		lipgloss.NewStyle().Bold(true).Render("GoTickets - Ticket Manager"), 
		len(m.storage.Tickets))
}

func (m *Model) filterList(query string) {
	if query == "" {
		m.refreshList()
		return
	}
	
	filteredTickets := m.storage.Search(query)
	items := make([]list.Item, len(filteredTickets))
	for i, ticket := range filteredTickets {
		items[i] = ticket
	}
	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("%s\nПоказано: %d из %d", 
		lipgloss.NewStyle().Bold(true).Render("GoTickets - Ticket Manager"), 
		len(filteredTickets), len(m.storage.Tickets))
}

func openBrowser(url string) func() error {
	return func() error {
		var cmd string
		var args []string

		switch runtime.GOOS {
		case "windows":
			cmd = "rundll32"
			args = []string{"url.dll,FileProtocolHandler", url}
		case "darwin":
			cmd = "open"
			args = []string{url}
		default: // "linux", "freebsd", "openbsd", "netbsd"
			cmd = "xdg-open"
			args = []string{url}
		}
		return exec.Command(cmd, args...).Start()
	}
}


func (m Model) updateAddURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.viewMode = ViewList
		m.textInput.SetValue("")
		m.textInput.Blur()
		m.tempURL = ""
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value != "" {
			m.tempURL = value
			m.viewMode = ViewAddTitle
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter ticket title..."
			m.textInput.Focus()
		}
		return m, nil
	}
	
	// Let textinput handle the input
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateAddTitle(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.viewMode = ViewList
		m.textInput.SetValue("")
		m.textInput.Blur()
		m.tempURL = ""
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value != "" {
			m.storage.AddTicket(value, m.tempURL)
			m.storage.Save()
			m.refreshList()
			m.viewMode = ViewList
			m.textInput.SetValue("")
			m.textInput.Blur()
			m.tempURL = ""
			// Move to the last item (newly added ticket)
			if len(m.storage.Tickets) > 0 {
				m.list.Select(len(m.storage.Tickets) - 1)
			}
		}
		return m, nil
	}
	
	// Let textinput handle the input
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.searchMode = false
		m.searchQuery = ""
		m.textInput.SetValue("")
		m.textInput.Blur()
		m.refreshList() // Reset to show all tickets
		return m, nil
	case "enter":
		// Apply current search and exit search mode
		m.searchMode = false
		m.searchQuery = m.textInput.Value()
		m.textInput.Blur()
		m.filterList(m.searchQuery)
		return m, nil
	case "up", "down", "k", "j":
		// Handle navigation within the list while in search mode
		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		return m, listCmd
	}
	
	// Let textinput handle the input and update search in real time
	m.textInput, cmd = m.textInput.Update(msg)
	
	// Filter tickets in real time as user types
	query := m.textInput.Value()
	m.filterList(query)
	
	return m, cmd
}

func (m Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "n":
		m.viewMode = ViewList
		m.ticketToDelete = -1
	case "y", "enter":
		if m.ticketToDelete != -1 {
			if m.storage.DeleteTicket(m.ticketToDelete) {
				m.storage.Save()
				m.refreshList()
			}
			m.viewMode = ViewList
			m.ticketToDelete = -1
		}
	}
	return m, nil
}

func (m Model) updateImport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.viewMode = ViewList
		m.textInput.SetValue("")
		m.textInput.Blur()
		return m, nil
	case "enter":
		filePath := strings.TrimSpace(m.textInput.Value())
		if filePath != "" {
			result, err := m.storage.ImportFromFile(filePath)
			if err != nil {
				// Показываем ошибку как результат с 0 добавленными тикетами
				m.importResult = &ImportResult{
					Added:      0,
					Duplicates: 0,
					Errors:     1,
					ErrorLines: []string{err.Error()},
				}
			} else {
				m.importResult = result
				// Сохраняем изменения, если были добавлены тикеты
				if result.Added > 0 {
					m.storage.Save()
					m.refreshList()
					// Navigate to last item to show newly imported tickets
					if len(m.storage.Tickets) > 0 {
						m.list.Select(len(m.storage.Tickets) - 1)
					}
				}
			}
			
			m.viewMode = ViewImportResult
			m.textInput.SetValue("")
			m.textInput.Blur()
		}
		return m, nil
	}
	
	// Let textinput handle the input
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateImportResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		m.viewMode = ViewList
		m.importResult = nil
		m.refreshList()
		return m, nil
	case "esc":
		m.viewMode = ViewList
		m.importResult = nil
		return m, nil
	}
	return m, nil
}

func (m Model) updateBackups(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.viewMode = ViewList
		m.selectedBackupIndex = -1
		return m, nil
	case "up", "k":
		if len(m.backups) > 0 {
			if m.selectedBackupIndex > 0 {
				m.selectedBackupIndex--
			} else {
				m.selectedBackupIndex = len(m.backups) - 1
			}
		}
		return m, nil
	case "down", "j":
		if len(m.backups) > 0 {
			if m.selectedBackupIndex < len(m.backups)-1 {
				m.selectedBackupIndex++
			} else {
				m.selectedBackupIndex = 0
			}
		}
		return m, nil
	case "enter":
		if len(m.backups) > 0 && m.selectedBackupIndex >= 0 && m.selectedBackupIndex < len(m.backups) {
			m.backupToRestore = m.backups[m.selectedBackupIndex]
			m.viewMode = ViewConfirmRestore
		}
		return m, nil
	case "esc":
		m.viewMode = ViewList
		m.selectedBackupIndex = -1
		return m, nil
	}
	return m, nil
}

func (m Model) updateConfirmRestore(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.viewMode = ViewList
		m.selectedBackupIndex = -1
		return m, nil
	case "y", "Y":
		// Confirm restore
		err := restoreFromBackup(m.backupToRestore)
		if err != nil {
			// TODO: Show error message
			return m, nil
		}
		// Reload tickets after restore
		storage, _ := LoadTickets()
		m.storage = storage
		m.refreshList()
		m.viewMode = ViewList
		m.selectedBackupIndex = -1
		return m, nil
	case "n", "N", "esc":
		m.viewMode = ViewList
		m.selectedBackupIndex = -1
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Padding(1, 2)

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	switch m.viewMode {
	case ViewList:
		if m.searchMode {
			// Show list with search input at bottom
			var s strings.Builder
			s.WriteString(m.list.View())
			s.WriteString("\n")
			s.WriteString(inputStyle.Render("Поиск: " + m.textInput.View()))
			s.WriteString("\n")
			s.WriteString("Enter - применить поиск, Esc - отмена")
			return s.String()
		} else {
			return m.list.View()
		}

	case ViewAddURL:
		s.WriteString(headerStyle.Render("Добавить новый тикет - Ссылка"))
		s.WriteString("\n\n")
		s.WriteString("Введите ссылку:\n")
		s.WriteString(inputStyle.Render(m.textInput.View()))
		s.WriteString("\n\n")
		s.WriteString("Enter - продолжить к названию, Esc - отмена\n")
	
	case ViewAddTitle:
		s.WriteString(headerStyle.Render("Добавить новый тикет - Название"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Ссылка: %s\n", m.tempURL))
		s.WriteString("Введите название тикета:\n")
		s.WriteString(inputStyle.Render(m.textInput.View()))
		s.WriteString("\n\n")
		s.WriteString("Enter - сохранить тикет, Esc - отмена\n")

	case ViewConfirmDelete:
		s.WriteString(headerStyle.Render("Подтверждение удаления"))
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
		s.WriteString(fmt.Sprintf("Тикет: #%d - %s\n\n", m.ticketToDelete, ticketTitle))
		s.WriteString("y/Enter - да, удалить\n")
		s.WriteString("n/Esc - нет, отменить\n")
	
	case ViewImport:
		s.WriteString(headerStyle.Render("Импорт тикетов"))
		s.WriteString("\n\n")
		s.WriteString("Введите путь к .txt файлу:\n")
		s.WriteString("Формат файла: каждая строка содержит 'URL - Название'\n\n")
		s.WriteString(inputStyle.Render(m.textInput.View()))
		s.WriteString("\n\n")
		s.WriteString("Enter - начать импорт, Esc - отмена\n")
	
	case ViewImportResult:
		s.WriteString(headerStyle.Render("Результат импорта"))
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
		
		s.WriteString("\nEnter/Esc/Пробел - вернуться к списку\n")
	
	case ViewBackups:
		s.WriteString(headerStyle.Render("Резервные копии"))
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
		s.WriteString("↑/↓ - навигация, Enter - выбрать, Esc - отмена\n")
	
	case ViewConfirmRestore:
		s.WriteString(headerStyle.Render("Подтверждение восстановления"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Вы уверены, что хотите восстановить из резервной копии?\n"))
		s.WriteString(fmt.Sprintf("Резервная копия: %s\n\n", m.backupToRestore))
		s.WriteString("⚠️  ВНИМАНИЕ: Это заменит все текущие тикеты!\n\n")
		s.WriteString("y/Enter - да, восстановить\n")
		s.WriteString("n/Esc - нет, отменить\n")
	}

	return s.String()
}