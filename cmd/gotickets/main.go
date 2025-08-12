package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"gotickets/pkg/gotickets"
)

func main() {
	p := tea.NewProgram(gotickets.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка запуска приложения: %v", err)
		os.Exit(1)
	}
}
