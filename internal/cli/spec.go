package cli

// Спецификация CLI: единственная точка истины, из которой строятся
// наборы флагов команд, текстовая справка и машиночитаемый манифест
// (`gotickets help --json`). Добавляя флаг, правьте только спецификацию.

// argSpec описывает позиционный аргумент команды
type argSpec struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // string | int
	Required    bool     `json:"required"`
	Repeated    bool     `json:"repeated,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description"`
}

// flagSpec описывает флаг команды
type flagSpec struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // bool | string | int
	Default     string   `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description"`
}

// commandSpec описывает одну команду CLI
type commandSpec struct {
	Name      string     `json:"name"`
	Aliases   []string   `json:"aliases,omitempty"`
	Summary   string     `json:"summary"`
	Usage     string     `json:"usage"`
	Args      []argSpec  `json:"args,omitempty"`
	Flags     []flagSpec `json:"flags,omitempty"`
	Output    string     `json:"output"`
	ExitCodes []int      `json:"exit_codes"`
	Examples  []string   `json:"examples,omitempty"`
}

// exitCodeSpec описывает код возврата процесса
type exitCodeSpec struct {
	Code        int    `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// manifest — полное машиночитаемое описание CLI
type manifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Summary     string         `json:"summary"`
	Usage       string         `json:"usage"`
	GlobalFlags []flagSpec     `json:"global_flags"`
	ExitCodes   []exitCodeSpec `json:"exit_codes"`
	Commands    []commandSpec  `json:"commands"`
	Notes       []string       `json:"notes"`
}

// jsonFlag доступен во всех командах и регистрируется автоматически
var jsonFlag = flagSpec{
	Name:        "json",
	Type:        "bool",
	Default:     "false",
	Description: "машиночитаемый вывод: данные JSON в stdout, {\"error\": \"...\"} в stderr",
}

var exitCodes = []exitCodeSpec{
	{ExitOK, "ok", "успех"},
	{ExitError, "error", "ошибка выполнения"},
	{ExitUsage, "usage", "неверные аргументы"},
	{ExitNotFound, "not_found", "тикет или бекап не найден"},
	{ExitDuplicate, "duplicate", "тикет с такой ссылкой уже существует"},
}

var notes = []string{
	"Запуск без аргументов открывает интерактивный TUI, с аргументами — неинтерактивный CLI.",
	"Текстовый вывод списков — TSV без заголовка: id<TAB>номер<TAB>название<TAB>ссылка.",
	"Данные в stdout, диагностика и ошибки в stderr; результат операции определяйте по коду возврата.",
	"Тикет адресуется по id или по точной ссылке (--url), поиск по ссылке регистрозависимый и точный.",
	"Перед каждой изменяющей операцией автоматически создается резервная копия.",
	"Хранилище общее с TUI: ~/.gotickets/tickets.json, путь можно получить командой path.",
}

var commandSpecs = []commandSpec{
	{
		Name:    "list",
		Aliases: []string{"ls"},
		Summary: "список тикетов",
		Usage:   "gotickets list [-q запрос] [-n N] [--json]",
		Flags: []flagSpec{
			{Name: "q", Type: "string", Description: "фильтр по названию и ссылке (подстрока, регистронезависимо)"},
			{Name: "n", Type: "int", Default: "0", Description: "максимальное количество тикетов, 0 — без ограничения"},
		},
		Output:    "массив тикетов",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage},
		Examples:  []string{"gotickets list --json", "gotickets list -q импорт -n 10"},
	},
	{
		Name:    "search",
		Aliases: []string{"find"},
		Summary: "поиск тикетов по названию и ссылке",
		Usage:   "gotickets search <запрос> [-n N] [--json]",
		Args: []argSpec{
			{Name: "query", Type: "string", Required: true, Repeated: true, Description: "поисковый запрос, несколько слов склеиваются пробелом"},
		},
		Flags: []flagSpec{
			{Name: "n", Type: "int", Default: "0", Description: "максимальное количество тикетов, 0 — без ограничения"},
		},
		Output:    "массив тикетов (пустой массив, если ничего не найдено)",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage},
		Examples:  []string{"gotickets search парсер --json"},
	},
	{
		Name:    "get",
		Aliases: []string{"show"},
		Summary: "получить один тикет по id или ссылке",
		Usage:   "gotickets get <id> | gotickets get --url <url>",
		Args: []argSpec{
			{Name: "id", Type: "int", Description: "идентификатор тикета; не указывается вместе с --url"},
		},
		Flags: []flagSpec{
			{Name: "url", Type: "string", Description: "искать тикет по точному совпадению ссылки вместо id"},
		},
		Output:    "объект тикета",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage, ExitNotFound},
		Examples:  []string{"gotickets get 12 --json", "gotickets get --url https://example.com/issues/42 --json"},
	},
	{
		Name:    "add",
		Aliases: []string{"new"},
		Summary: "добавить тикет",
		Usage:   "gotickets add <url> <название> [--upsert] [--allow-duplicate] [--json]",
		Args: []argSpec{
			{Name: "url", Type: "string", Description: "ссылка на тикет; альтернатива флагу -u"},
			{Name: "title", Type: "string", Repeated: true, Description: "название тикета; альтернатива флагу -t"},
		},
		Flags: []flagSpec{
			{Name: "u", Type: "string", Description: "ссылка на тикет"},
			{Name: "t", Type: "string", Description: "название тикета"},
			{Name: "upsert", Type: "bool", Default: "false", Description: "если тикет с такой ссылкой есть — обновить название и вернуть его с кодом 0"},
			{Name: "allow-duplicate", Type: "bool", Default: "false", Description: "разрешить второй тикет с уже существующей ссылкой"},
		},
		Output:    "объект тикета; при дубликате в stdout уходит существующий тикет, а код возврата равен 4",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage, ExitDuplicate},
		Examples: []string{
			"gotickets add https://example.com/issues/42 \"Починить импорт\" --json",
			"gotickets add -u https://example.com/issues/42 -t \"Починить импорт\" --upsert --json",
		},
	},
	{
		Name:    "update",
		Aliases: []string{"edit"},
		Summary: "изменить ссылку и/или название тикета",
		Usage:   "gotickets update <id> [-u url] [-t название] [--json]",
		Args: []argSpec{
			{Name: "id", Type: "int", Required: true, Description: "идентификатор тикета"},
		},
		Flags: []flagSpec{
			{Name: "u", Type: "string", Description: "новая ссылка; пустое значение оставляет прежнюю"},
			{Name: "t", Type: "string", Description: "новое название; пустое значение оставляет прежнее"},
		},
		Output:    "объект обновленного тикета",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage, ExitNotFound},
		Examples:  []string{"gotickets update 12 -t \"Новое название\" --json"},
	},
	{
		Name:    "rm",
		Aliases: []string{"delete", "remove"},
		Summary: "удалить тикеты без подтверждения",
		Usage:   "gotickets rm <id> [id...] | gotickets rm --url <url>",
		Args: []argSpec{
			{Name: "id", Type: "int", Repeated: true, Description: "идентификаторы тикетов; не указываются вместе с --url"},
		},
		Flags: []flagSpec{
			{Name: "url", Type: "string", Description: "удалить тикет по точному совпадению ссылки вместо id"},
		},
		Output:    "массив удаленных тикетов; при несуществующем id не удаляется ничего",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage, ExitNotFound},
		Examples:  []string{"gotickets rm 1 2 3 --json", "gotickets rm --url https://example.com/issues/42"},
	},
	{
		Name:    "open",
		Summary: "открыть ссылку тикета в браузере",
		Usage:   "gotickets open <id> [--print] [--json]",
		Args: []argSpec{
			{Name: "id", Type: "int", Required: true, Description: "идентификатор тикета"},
		},
		Flags: []flagSpec{
			{Name: "print", Type: "bool", Default: "false", Description: "только напечатать ссылку, не запускать браузер"},
		},
		Output:    "объект {\"url\": \"...\"}",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage, ExitNotFound},
		Examples:  []string{"gotickets open 12 --print"},
	},
	{
		Name:    "import",
		Summary: "импорт тикетов из txt-файла",
		Usage:   "gotickets import <файл> [--json]",
		Args: []argSpec{
			{Name: "file", Type: "string", Required: true, Description: "файл со строками вида \"URL - Название\""},
		},
		Output:    "объект {\"added\", \"duplicates\", \"errors\", \"error_lines\"}",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage},
		Examples:  []string{"gotickets import tickets.txt --json"},
	},
	{
		Name:    "export",
		Summary: "выгрузить все тикеты",
		Usage:   "gotickets export [-f txt|json] [-o файл]",
		Flags: []flagSpec{
			{Name: "f", Type: "string", Default: "txt", Enum: []string{"txt", "json"}, Description: "формат вывода; txt совместим с командой import"},
			{Name: "o", Type: "string", Description: "файл для записи; по умолчанию stdout"},
		},
		Output:    "строки \"URL - Название\" или массив тикетов",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage},
		Examples:  []string{"gotickets export -o tickets.txt", "gotickets export -f json"},
	},
	{
		Name:    "backup",
		Summary: "работа с резервными копиями",
		Usage:   "gotickets backup list|create|restore <имя> [--json]",
		Args: []argSpec{
			{Name: "subcommand", Type: "string", Required: true, Enum: []string{"list", "create", "restore"}, Description: "действие с резервными копиями"},
			{Name: "name", Type: "string", Description: "имя резервной копии, только для restore"},
		},
		Output:    "массив имен файлов (list) либо объект с полем created/restored",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage, ExitNotFound},
		Examples:  []string{"gotickets backup create --json", "gotickets backup restore tickets_backup_2026-01-01_10-00-00.json"},
	},
	{
		Name:      "path",
		Summary:   "пути к директории и файлу данных",
		Usage:     "gotickets path [--json]",
		Output:    "объект {\"dir\": \"...\", \"file\": \"...\"}",
		ExitCodes: []int{ExitOK, ExitError, ExitUsage},
	},
	{
		Name:      "version",
		Aliases:   []string{"-v", "--version"},
		Summary:   "версия приложения",
		Usage:     "gotickets version",
		Output:    "строка с версией",
		ExitCodes: []int{ExitOK},
	},
	{
		Name:      "help",
		Aliases:   []string{"-h", "--help"},
		Summary:   "справка; с --json отдает полное описание CLI",
		Usage:     "gotickets help [--json]",
		Output:    "текстовая справка либо манифест CLI в JSON",
		ExitCodes: []int{ExitOK},
		Examples:  []string{"gotickets help --json"},
	},
}

// lookupSpec находит команду по имени или алиасу
func lookupSpec(name string) *commandSpec {
	for i := range commandSpecs {
		if commandSpecs[i].Name == name {
			return &commandSpecs[i]
		}
		for _, alias := range commandSpecs[i].Aliases {
			if alias == name {
				return &commandSpecs[i]
			}
		}
	}
	return nil
}

// mustSpec возвращает спецификацию команды; отсутствие означает ошибку в коде
func mustSpec(name string) *commandSpec {
	spec := lookupSpec(name)
	if spec == nil {
		panic("нет спецификации для команды " + name)
	}
	return spec
}

func newManifest() manifest {
	return manifest{
		Name:        "gotickets",
		Version:     Version,
		Summary:     "менеджер тикетов: интерактивный TUI и неинтерактивный CLI поверх одного хранилища",
		Usage:       "gotickets <команда> [аргументы] [флаги]",
		GlobalFlags: []flagSpec{jsonFlag},
		ExitCodes:   exitCodes,
		Commands:    commandSpecs,
		Notes:       notes,
	}
}
