package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"gotickets/internal/cli"
	"gotickets/pkg/gotickets"
)

func main() {
	// С аргументами работаем как обычная CLI-утилита, без TUI
	if cli.IsCommand(os.Args[1:]) {
		os.Exit(cli.Run(os.Args[1:]))
	}

	p := tea.NewProgram(gotickets.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка запуска приложения: %v", err)
		os.Exit(1)
	}
}
