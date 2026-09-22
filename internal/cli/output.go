package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"gotickets/internal/storage"
)

// ticketOut — представление тикета для вывода наружу.
// Отдельная структура нужна, чтобы отдавать вычисляемый номер тикета.
type ticketOut struct {
	ID        int       `json:"id"`
	Number    string    `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

func toOut(t storage.Ticket) ticketOut {
	return ticketOut{
		ID:        t.ID,
		Number:    t.ExtractTicketNumber(),
		Title:     t.Title,
		URL:       t.URL,
		CreatedAt: t.CreatedAt,
	}
}

func toOutSlice(tickets []storage.Ticket) []ticketOut {
	out := make([]ticketOut, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, toOut(t))
	}
	return out
}

// values дает типизированный доступ к разобранным флагам по имени из спецификации
type values struct{ fs *flag.FlagSet }

func (v values) lookup(name string) flag.Getter {
	f := v.fs.Lookup(name)
	if f == nil {
		panic("флаг -" + name + " не объявлен в спецификации команды " + v.fs.Name())
	}
	return f.Value.(flag.Getter)
}

func (v values) Bool(name string) bool     { return v.lookup(name).Get().(bool) }
func (v values) String(name string) string { return v.lookup(name).Get().(string) }
func (v values) Int(name string) int       { return v.lookup(name).Get().(int) }

// newFlagSet строит набор флагов команды по ее спецификации,
// добавляя общий для всех команд флаг --json
func newFlagSet(spec *commandSpec) (*flag.FlagSet, values) {
	fs := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	for _, f := range append([]flagSpec{jsonFlag}, spec.Flags...) {
		switch f.Type {
		case "bool":
			fs.Bool(f.Name, f.Default == "true", f.Description)
		case "int":
			def, _ := strconv.Atoi(f.Default)
			fs.Int(f.Name, def, f.Description)
		default:
			fs.String(f.Name, f.Default, f.Description)
		}
	}
	return fs, values{fs}
}

// parseFlags разбирает аргументы команды, допуская флаги после позиционных
// аргументов (стандартный flag на первом позиционном останавливается).
// Возвращает позиционные аргументы и признак успеха.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, bool) {
	var tail []string
	// Всё после "--" считается позиционными аргументами
	for i, a := range args {
		if a == "--" {
			tail = append(tail, args[i+1:]...)
			args = args[:i]
			break
		}
	}

	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	return append(positional, tail...), true
}

func printJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// printTickets печатает тикеты: JSON-массивом или TSV-строками
func printTickets(tickets []ticketOut, jsonOut bool) int {
	if jsonOut {
		if err := printJSON(os.Stdout, tickets); err != nil {
			return fail(jsonOut, ExitError, "не удалось сериализовать вывод: %v", err)
		}
		return ExitOK
	}
	for _, t := range tickets {
		fmt.Printf("%d\t%s\t%s\t%s\n", t.ID, t.Number, sanitize(t.Title), sanitize(t.URL))
	}
	return ExitOK
}

// printTicket печатает один тикет: JSON-объектом или TSV-строкой
func printTicket(t ticketOut, jsonOut bool) {
	if jsonOut {
		_ = printJSON(os.Stdout, t)
		return
	}
	fmt.Printf("%d\t%s\t%s\t%s\n", t.ID, t.Number, sanitize(t.Title), sanitize(t.URL))
}

// sanitize убирает символы, ломающие построчный TSV-вывод
func sanitize(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}

// fail печатает ошибку в stderr и возвращает код возврата
func fail(jsonOut bool, code int, format string, a ...interface{}) int {
	msg := fmt.Sprintf(format, a...)
	if jsonOut {
		_ = printJSON(os.Stderr, map[string]string{"error": msg})
	} else {
		fmt.Fprintf(os.Stderr, "ошибка: %s\n", msg)
	}
	return code
}

// ok печатает результат операции: JSON-объект или человекочитаемую строку
func ok(jsonOut bool, payload interface{}, format string, a ...interface{}) int {
	if jsonOut {
		if err := printJSON(os.Stdout, payload); err != nil {
			return fail(jsonOut, ExitError, "не удалось сериализовать вывод: %v", err)
		}
		return ExitOK
	}
	fmt.Printf(format+"\n", a...)
	return ExitOK
}
