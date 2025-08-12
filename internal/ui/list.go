package ui

import (
	"fmt"
	"io"

	"gotickets/internal/storage"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// ticketDelegate implements the list.ItemDelegate interface for rendering tickets
type ticketDelegate struct{}

func (d ticketDelegate) Height() int                             { return 1 }
func (d ticketDelegate) Spacing() int                            { return 0 }
func (d ticketDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d ticketDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	ticket, ok := listItem.(storage.Ticket)
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
			Render("> "+str))
	} else {
		fmt.Fprint(w, lipgloss.NewStyle().PaddingLeft(4).Render(str))
	}
}

// createList creates and configures the main ticket list
func createList(items []list.Item, ticketCount int) list.Model {
	l := list.New(items, ticketDelegate{}, 80, 24)
	l.Title = fmt.Sprintf("%s\nВсего тикетов: %d",
		lipgloss.NewStyle().Bold(true).Render("GoTickets - Ticket Manager"),
		ticketCount)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // We'll handle search manually
	l.Styles.Title = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Align(lipgloss.Center)

	// Set additional help keys
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

	return l
}

// RefreshList updates the list with current tickets
func (m *Model) RefreshList() {
	items := make([]list.Item, len(m.storage.Tickets))
	for i, ticket := range m.storage.Tickets {
		items[i] = ticket
	}
	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("%s\nВсего тикетов: %d",
		lipgloss.NewStyle().Bold(true).Render("GoTickets - Ticket Manager"),
		len(m.storage.Tickets))
}

// FilterList filters the list based on query
func (m *Model) FilterList(query string) {
	if query == "" {
		m.RefreshList()
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