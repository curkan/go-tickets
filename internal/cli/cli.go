// Package cli реализует неинтерактивный режим работы приложения:
// одна команда — один запуск, вывод в stdout, ошибки в stderr, осмысленные коды возврата.
// Режим предназначен для скриптов и агентов, которым не нужен TUI.
package cli

import (
	"fmt"
	"os"
	"strings"
)

// Version подставляется при сборке через -ldflags "-X gotickets/internal/cli.Version=..."
var Version = "dev"

// Коды возврата
const (
	ExitOK        = 0 // успех
	ExitError     = 1 // ошибка выполнения
	ExitUsage     = 2 // неверные аргументы
	ExitNotFound  = 3 // тикет/бекап не найден
	ExitDuplicate = 4 // тикет с такой ссылкой уже существует
)

// IsCommand сообщает, нужно ли запускать CLI вместо TUI
func IsCommand(args []string) bool { return len(args) > 0 }

// Run выполняет одну команду и возвращает код возврата процесса
func Run(args []string) int {
	if len(args) == 0 {
		usage(os.Stderr)
		return ExitUsage
	}

	name, rest := args[0], args[1:]
	spec := lookupSpec(name)
	if spec == nil {
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", name)
		usage(os.Stderr)
		return ExitUsage
	}

	switch spec.Name {
	case "help":
		return cmdHelp(rest)
	case "version":
		fmt.Println(Version)
		return ExitOK
	case "list":
		return cmdList(rest)
	case "search":
		return cmdSearch(rest)
	case "get":
		return cmdGet(rest)
	case "add":
		return cmdAdd(rest)
	case "update":
		return cmdUpdate(rest)
	case "rm":
		return cmdRemove(rest)
	case "open":
		return cmdOpen(rest)
	case "import":
		return cmdImport(rest)
	case "export":
		return cmdExport(rest)
	case "backup":
		return cmdBackup(rest)
	case "path":
		return cmdPath(rest)
	default:
		panic("команда " + spec.Name + " описана в спецификации, но не реализована")
	}
}

// usage печатает текстовую справку, собранную из спецификации команд
func usage(f *os.File) {
	m := newManifest()

	fmt.Fprintf(f, "%s — %s.\n\n", m.Name, m.Summary)
	fmt.Fprintf(f, "Использование:\n  %s\n\nКоманды:\n", m.Usage)

	// Короткие строки использования выравниваем в колонку, длинные переносим
	const column = 44
	for _, cmd := range m.Commands {
		line := strings.TrimPrefix(cmd.Usage, "gotickets ")
		if len(line) > column {
			fmt.Fprintf(f, "  %s\n  %s%s\n", line, strings.Repeat(" ", column), cmd.Summary)
			continue
		}
		fmt.Fprintf(f, "  %-*s%s\n", column, line, cmd.Summary)
	}

	fmt.Fprintf(f, "\nОбщие флаги:\n")
	for _, fl := range m.GlobalFlags {
		fmt.Fprintf(f, "  %-10s%s\n", "--"+fl.Name, fl.Description)
	}

	fmt.Fprintf(f, "\nКоды возврата:\n")
	for _, code := range m.ExitCodes {
		fmt.Fprintf(f, "  %-10d%s\n", code.Code, code.Description)
	}

	fmt.Fprintf(f, "\nОсобенности:\n")
	for _, note := range m.Notes {
		fmt.Fprintf(f, "  - %s\n", note)
	}

	fmt.Fprintf(f, "\nПолное машиночитаемое описание CLI: gotickets help --json\n")
}
