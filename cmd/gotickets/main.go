package main

import (
	"fmt"
	"os"

	"gotickets/pkg/gotickets"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(gotickets.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка запуска приложения: %v", err)
		os.Exit(1)
	}
}