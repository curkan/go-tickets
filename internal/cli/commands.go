package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gotickets/internal/browser"
	"gotickets/internal/storage"
)

func fs() storage.FileSystem { return &storage.RealFileSystem{} }

func load() (*storage.TicketStorage, error) {
	return storage.LoadTicketsWithFS(fs())
}

// begin — общее начало команды: разбор флагов по спецификации и загрузка хранилища
func begin(name string, args []string) (*storage.TicketStorage, []string, values, int) {
	fset, vals := newFlagSet(mustSpec(name))
	rest, okParse := parseFlags(fset, args)
	if !okParse {
		return nil, nil, vals, ExitUsage
	}
	ts, err := load()
	if err != nil {
		return nil, nil, vals, fail(vals.Bool("json"), ExitError, "не удалось загрузить тикеты: %v", err)
	}
	return ts, rest, vals, ExitOK
}

// limitTickets обрезает выборку, n <= 0 означает "без ограничения"
func limitTickets(tickets []storage.Ticket, n int) []storage.Ticket {
	if n > 0 && len(tickets) > n {
		return tickets[:n]
	}
	return tickets
}

// findByURL ищет тикет по точному совпадению ссылки
func findByURL(ts *storage.TicketStorage, url string) (storage.Ticket, bool) {
	for _, ticket := range ts.Tickets {
		if ticket.URL == url {
			return ticket, true
		}
	}
	return storage.Ticket{}, false
}

// parseID разбирает позиционный идентификатор тикета
func parseID(arg string, jsonOut bool) (int, int) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fail(jsonOut, ExitUsage, "id должен быть числом: %s", arg)
	}
	return id, ExitOK
}

func cmdList(args []string) int {
	ts, rest, vals, code := begin("list", args)
	if ts == nil {
		return code
	}
	if len(rest) > 0 {
		return fail(vals.Bool("json"), ExitUsage, "list не принимает позиционных аргументов, используйте -q")
	}
	return printTickets(toOutSlice(limitTickets(ts.Search(vals.String("q")), vals.Int("n"))), vals.Bool("json"))
}

func cmdSearch(args []string) int {
	ts, rest, vals, code := begin("search", args)
	if ts == nil {
		return code
	}
	if len(rest) == 0 {
		return fail(vals.Bool("json"), ExitUsage, "укажите поисковый запрос: gotickets search <запрос>")
	}
	found := ts.Search(strings.Join(rest, " "))
	return printTickets(toOutSlice(limitTickets(found, vals.Int("n"))), vals.Bool("json"))
}

func cmdGet(args []string) int {
	ts, rest, vals, code := begin("get", args)
	if ts == nil {
		return code
	}
	jsonOut, url := vals.Bool("json"), vals.String("url")

	var ticket storage.Ticket
	var found bool
	switch {
	case url != "" && len(rest) > 0:
		return fail(jsonOut, ExitUsage, "укажите либо id, либо --url, но не оба сразу")
	case url != "":
		ticket, found = findByURL(ts, url)
		if !found {
			return fail(jsonOut, ExitNotFound, "тикет со ссылкой %s не найден", url)
		}
	case len(rest) == 1:
		id, code := parseID(rest[0], jsonOut)
		if code != ExitOK {
			return code
		}
		if ticket, found = ts.GetByID(id); !found {
			return fail(jsonOut, ExitNotFound, "тикет %d не найден", id)
		}
	default:
		return fail(jsonOut, ExitUsage, "укажите id тикета или ссылку: gotickets get <id> | gotickets get --url <url>")
	}

	out := toOut(ticket)
	return ok(jsonOut, out, "%d\t%s\t%s\t%s", out.ID, out.Number, sanitize(out.Title), sanitize(out.URL))
}

func cmdAdd(args []string) int {
	ts, rest, vals, code := begin("add", args)
	if ts == nil {
		return code
	}
	jsonOut := vals.Bool("json")
	url, title := vals.String("u"), vals.String("t")

	// Позиционная форма: add <url> <название...>
	if url == "" && len(rest) > 0 {
		url, rest = rest[0], rest[1:]
	}
	if title == "" && len(rest) > 0 {
		title, rest = strings.Join(rest, " "), nil
	}
	if len(rest) > 0 {
		return fail(jsonOut, ExitUsage, "лишние аргументы: %s", strings.Join(rest, " "))
	}
	if url == "" || title == "" {
		return fail(jsonOut, ExitUsage, "нужны ссылка и название: gotickets add <url> <название>")
	}

	// Дубликат по ссылке: отдаем существующий тикет, чтобы вызывающему
	// не пришлось делать второй запрос ради его id
	if existing, found := findByURL(ts, url); found && !vals.Bool("allow-duplicate") {
		if !vals.Bool("upsert") {
			printTicket(toOut(existing), jsonOut)
			return fail(jsonOut, ExitDuplicate, "тикет с такой ссылкой уже существует: %s", url)
		}
		updated, _ := ts.UpdateTicket(existing.ID, title, "")
		if err := ts.Save(); err != nil {
			return fail(jsonOut, ExitError, "не удалось сохранить тикеты: %v", err)
		}
		out := toOut(updated)
		return ok(jsonOut, out, "обновлен тикет %d\t%s\t%s\t%s", out.ID, out.Number, sanitize(out.Title), sanitize(out.URL))
	}

	ts.AddTicket(title, url)
	if err := ts.Save(); err != nil {
		return fail(jsonOut, ExitError, "не удалось сохранить тикеты: %v", err)
	}
	added := toOut(ts.Tickets[len(ts.Tickets)-1])
	return ok(jsonOut, added, "добавлен тикет %d\t%s\t%s\t%s", added.ID, added.Number, sanitize(added.Title), sanitize(added.URL))
}

func cmdUpdate(args []string) int {
	ts, rest, vals, code := begin("update", args)
	if ts == nil {
		return code
	}
	jsonOut := vals.Bool("json")
	if len(rest) != 1 {
		return fail(jsonOut, ExitUsage, "укажите id тикета: gotickets update <id> [-u url] [-t название]")
	}
	id, code := parseID(rest[0], jsonOut)
	if code != ExitOK {
		return code
	}
	url, title := vals.String("u"), vals.String("t")
	if url == "" && title == "" {
		return fail(jsonOut, ExitUsage, "укажите хотя бы один из флагов -u или -t")
	}

	updated, found := ts.UpdateTicket(id, title, url)
	if !found {
		return fail(jsonOut, ExitNotFound, "тикет %d не найден", id)
	}
	if err := ts.Save(); err != nil {
		return fail(jsonOut, ExitError, "не удалось сохранить тикеты: %v", err)
	}
	out := toOut(updated)
	return ok(jsonOut, out, "обновлен тикет %d\t%s\t%s\t%s", out.ID, out.Number, sanitize(out.Title), sanitize(out.URL))
}

func cmdRemove(args []string) int {
	ts, rest, vals, code := begin("rm", args)
	if ts == nil {
		return code
	}
	jsonOut, url := vals.Bool("json"), vals.String("url")

	var ids []int
	switch {
	case url != "" && len(rest) > 0:
		return fail(jsonOut, ExitUsage, "укажите либо id, либо --url, но не оба сразу")
	case url != "":
		ticket, found := findByURL(ts, url)
		if !found {
			return fail(jsonOut, ExitNotFound, "тикет со ссылкой %s не найден", url)
		}
		ids = []int{ticket.ID}
	case len(rest) > 0:
		ids = make([]int, 0, len(rest))
		for _, arg := range rest {
			id, code := parseID(arg, jsonOut)
			if code != ExitOK {
				return code
			}
			ids = append(ids, id)
		}
	default:
		return fail(jsonOut, ExitUsage, "укажите id тикетов или ссылку: gotickets rm <id> [id...] | gotickets rm --url <url>")
	}

	// Проверяем все id заранее: удаление либо проходит целиком, либо не начинается
	for _, id := range ids {
		if _, found := ts.GetByID(id); !found {
			return fail(jsonOut, ExitNotFound, "тикет %d не найден", id)
		}
	}

	deleted := make([]ticketOut, 0, len(ids))
	for _, id := range ids {
		ticket, _ := ts.GetByID(id)
		if ts.DeleteTicket(id) {
			deleted = append(deleted, toOut(ticket))
		}
	}
	if err := ts.Save(); err != nil {
		return fail(jsonOut, ExitError, "не удалось сохранить тикеты: %v", err)
	}
	if jsonOut {
		return printTickets(deleted, true)
	}
	fmt.Printf("удалено тикетов: %d\n", len(deleted))
	return ExitOK
}

func cmdOpen(args []string) int {
	ts, rest, vals, code := begin("open", args)
	if ts == nil {
		return code
	}
	jsonOut := vals.Bool("json")
	if len(rest) != 1 {
		return fail(jsonOut, ExitUsage, "укажите id тикета: gotickets open <id>")
	}
	id, code := parseID(rest[0], jsonOut)
	if code != ExitOK {
		return code
	}
	ticket, found := ts.GetByID(id)
	if !found {
		return fail(jsonOut, ExitNotFound, "тикет %d не найден", id)
	}
	if !vals.Bool("print") {
		if err := browser.Open(ticket.URL); err != nil {
			return fail(jsonOut, ExitError, "не удалось открыть браузер: %v", err)
		}
	}
	return ok(jsonOut, map[string]string{"url": ticket.URL}, "%s", ticket.URL)
}

func cmdImport(args []string) int {
	ts, rest, vals, code := begin("import", args)
	if ts == nil {
		return code
	}
	jsonOut := vals.Bool("json")
	if len(rest) != 1 {
		return fail(jsonOut, ExitUsage, "укажите файл: gotickets import <файл>")
	}

	result, err := ts.ImportFromFile(rest[0])
	if err != nil {
		return fail(jsonOut, ExitError, "%v", err)
	}
	if err := ts.Save(); err != nil {
		return fail(jsonOut, ExitError, "не удалось сохранить тикеты: %v", err)
	}
	if jsonOut {
		return ok(true, result, "")
	}
	fmt.Printf("добавлено: %d, дубликатов: %d, ошибок: %d\n", result.Added, result.Duplicates, result.Errors)
	for _, line := range result.ErrorLines {
		fmt.Fprintln(os.Stderr, line)
	}
	return ExitOK
}

func cmdExport(args []string) int {
	ts, rest, vals, code := begin("export", args)
	if ts == nil {
		return code
	}
	jsonOut := vals.Bool("json")
	if len(rest) > 0 {
		return fail(jsonOut, ExitUsage, "export не принимает позиционных аргументов")
	}
	format := vals.String("f")
	if jsonOut {
		format = "json"
	}
	if format != "txt" && format != "json" {
		return fail(jsonOut, ExitUsage, "неизвестный формат: %s (ожидается txt или json)", format)
	}

	var buf strings.Builder
	if format == "json" {
		if err := printJSON(&buf, toOutSlice(ts.Tickets)); err != nil {
			return fail(jsonOut, ExitError, "не удалось сериализовать вывод: %v", err)
		}
	} else {
		// Формат, совместимый с командой import
		for _, t := range ts.Tickets {
			fmt.Fprintf(&buf, "%s - %s\n", sanitize(t.URL), sanitize(t.Title))
		}
	}

	out := vals.String("o")
	if out == "" {
		fmt.Print(buf.String())
		return ExitOK
	}
	if err := os.WriteFile(out, []byte(buf.String()), 0644); err != nil {
		return fail(jsonOut, ExitError, "не удалось записать файл: %v", err)
	}
	return ExitOK
}

func cmdBackup(args []string) int {
	if len(args) == 0 {
		return fail(false, ExitUsage, "укажите подкоманду: gotickets backup list|create|restore <имя>")
	}
	sub := args[0]

	fset, vals := newFlagSet(mustSpec("backup"))
	pos, okParse := parseFlags(fset, args[1:])
	if !okParse {
		return ExitUsage
	}
	jsonOut := vals.Bool("json")

	switch sub {
	case "list", "ls":
		if len(pos) > 0 {
			return fail(jsonOut, ExitUsage, "backup list не принимает аргументов")
		}
		backups, err := storage.ListBackupsUsing(fs())
		if err != nil {
			return fail(jsonOut, ExitError, "не удалось получить список бекапов: %v", err)
		}
		if jsonOut {
			if backups == nil {
				backups = []string{}
			}
			return ok(true, backups, "")
		}
		for _, name := range backups {
			fmt.Println(name)
		}
		return ExitOK

	case "create":
		if err := storage.CreateBackupUsing(fs()); err != nil {
			return fail(jsonOut, ExitError, "не удалось создать бекап: %v", err)
		}
		// Имя свежего бекапа определяем по последнему в списке
		backups, err := storage.ListBackupsUsing(fs())
		if err != nil || len(backups) == 0 {
			return ok(jsonOut, map[string]string{"created": ""}, "бекап создан")
		}
		latest := backups[len(backups)-1]
		return ok(jsonOut, map[string]string{"created": latest}, "бекап создан: %s", latest)

	case "restore":
		if len(pos) != 1 {
			return fail(jsonOut, ExitUsage, "укажите имя бекапа: gotickets backup restore <имя>")
		}
		name := pos[0]
		if err := storage.RestoreFromBackupUsing(fs(), name); err != nil {
			if strings.Contains(err.Error(), "backup file not found") {
				return fail(jsonOut, ExitNotFound, "бекап не найден: %s", name)
			}
			return fail(jsonOut, ExitError, "%v", err)
		}
		return ok(jsonOut, map[string]string{"restored": name}, "восстановлено из бекапа: %s", name)

	default:
		return fail(jsonOut, ExitUsage, "неизвестная подкоманда backup: %s", sub)
	}
}

func cmdPath(args []string) int {
	fset, vals := newFlagSet(mustSpec("path"))
	rest, okParse := parseFlags(fset, args)
	if !okParse {
		return ExitUsage
	}
	jsonOut := vals.Bool("json")
	if len(rest) > 0 {
		return fail(jsonOut, ExitUsage, "path не принимает аргументов")
	}
	dir, err := storage.DataDirUsing(fs())
	if err != nil {
		return fail(jsonOut, ExitError, "не удалось определить домашнюю директорию: %v", err)
	}
	file, err := storage.DataFilePathUsing(fs())
	if err != nil {
		return fail(jsonOut, ExitError, "не удалось определить путь к данным: %v", err)
	}
	return ok(jsonOut, map[string]string{"dir": dir, "file": file}, "%s", file)
}

func cmdHelp(args []string) int {
	fset, vals := newFlagSet(mustSpec("help"))
	if _, okParse := parseFlags(fset, args); !okParse {
		return ExitUsage
	}
	if vals.Bool("json") {
		if err := printJSON(os.Stdout, newManifest()); err != nil {
			return fail(true, ExitError, "не удалось сериализовать манифест: %v", err)
		}
		return ExitOK
	}
	usage(os.Stdout)
	return ExitOK
}
