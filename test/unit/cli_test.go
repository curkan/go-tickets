package unit

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"gotickets/internal/cli"
)

// runCLI выполняет команду в изолированном HOME и возвращает stdout, stderr и код возврата
func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("не удалось создать pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("не удалось создать pipe: %v", err)
	}

	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	code := cli.Run(args)
	os.Stdout, os.Stderr = origOut, origErr

	outW.Close()
	errW.Close()
	stdout, _ := io.ReadAll(outR)
	stderr, _ := io.ReadAll(errR)
	return string(stdout), string(stderr), code
}

// withIsolatedHome перенаправляет домашнюю директорию во временную,
// чтобы тесты не трогали реальные данные пользователя
func withIsolatedHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestCLIAddListGet(t *testing.T) {
	withIsolatedHome(t)

	stdout, _, code := runCLI(t, "add", "https://example.com/issues/42", "Test ticket", "--json")
	if code != cli.ExitOK {
		t.Fatalf("add: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	var added map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil {
		t.Fatalf("add: невалидный JSON: %v (%q)", err, stdout)
	}
	if added["title"] != "Test ticket" || added["number"] != "42" {
		t.Errorf("add: неожиданный тикет: %v", added)
	}

	stdout, _, code = runCLI(t, "list")
	if code != cli.ExitOK {
		t.Fatalf("list: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	fields := strings.Split(strings.TrimSpace(stdout), "\t")
	if len(fields) != 4 || fields[0] != "1" || fields[3] != "https://example.com/issues/42" {
		t.Errorf("list: неожиданный TSV-вывод: %q", stdout)
	}

	stdout, _, code = runCLI(t, "get", "1", "--json")
	if code != cli.ExitOK {
		t.Fatalf("get: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	if !strings.Contains(stdout, "\"id\": 1") {
		t.Errorf("get: ожидался тикет с id=1, получено %q", stdout)
	}
}

func TestCLIDuplicateAndNotFound(t *testing.T) {
	withIsolatedHome(t)

	if _, _, code := runCLI(t, "add", "https://example.com/1", "One"); code != cli.ExitOK {
		t.Fatalf("add: ожидался успех, получен код %d", code)
	}
	if _, _, code := runCLI(t, "add", "https://example.com/1", "One again"); code != cli.ExitDuplicate {
		t.Errorf("add дубликата: ожидался код %d, получен %d", cli.ExitDuplicate, code)
	}
	if _, _, code := runCLI(t, "add", "https://example.com/1", "One again", "--allow-duplicate"); code != cli.ExitOK {
		t.Errorf("add --allow-duplicate: ожидался код %d, получен %d", cli.ExitOK, code)
	}

	_, stderr, code := runCLI(t, "get", "999", "--json")
	if code != cli.ExitNotFound {
		t.Errorf("get несуществующего: ожидался код %d, получен %d", cli.ExitNotFound, code)
	}
	if !strings.Contains(stderr, "\"error\"") {
		t.Errorf("get несуществующего: ошибка должна быть JSON в stderr, получено %q", stderr)
	}
}

func TestCLIUpdateAndRemove(t *testing.T) {
	withIsolatedHome(t)

	runCLI(t, "add", "https://example.com/1", "One")
	runCLI(t, "add", "https://example.com/2", "Two")

	stdout, _, code := runCLI(t, "update", "1", "-t", "Renamed", "--json")
	if code != cli.ExitOK {
		t.Fatalf("update: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	if !strings.Contains(stdout, "Renamed") {
		t.Errorf("update: название не изменилось: %q", stdout)
	}

	// Удаление не должно выполняться частично: несуществующий id отменяет всю операцию
	if _, _, code := runCLI(t, "rm", "1", "999"); code != cli.ExitNotFound {
		t.Errorf("rm с несуществующим id: ожидался код %d, получен %d", cli.ExitNotFound, code)
	}
	stdout, _, _ = runCLI(t, "list")
	if lines := strings.Count(strings.TrimSpace(stdout), "\n") + 1; lines != 2 {
		t.Errorf("rm: тикеты не должны были удалиться, осталось строк: %d", lines)
	}

	if _, _, code := runCLI(t, "rm", "1", "2"); code != cli.ExitOK {
		t.Errorf("rm: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	stdout, _, _ = runCLI(t, "list")
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("rm: список должен быть пустым, получено %q", stdout)
	}
}

func TestCLISearchAndExport(t *testing.T) {
	withIsolatedHome(t)

	runCLI(t, "add", "https://example.com/1", "Alpha bug")
	runCLI(t, "add", "https://example.com/2", "Beta feature")

	stdout, _, code := runCLI(t, "search", "alpha")
	if code != cli.ExitOK {
		t.Fatalf("search: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	if strings.Count(strings.TrimSpace(stdout), "\n") != 0 || !strings.Contains(stdout, "Alpha bug") {
		t.Errorf("search: ожидался ровно один результат, получено %q", stdout)
	}

	// Текстовый экспорт совместим с форматом импорта
	stdout, _, code = runCLI(t, "export")
	if code != cli.ExitOK {
		t.Fatalf("export: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	if !strings.Contains(stdout, "https://example.com/1 - Alpha bug") {
		t.Errorf("export: неожиданный формат: %q", stdout)
	}
}

func TestCLIImport(t *testing.T) {
	withIsolatedHome(t)

	file := t.TempDir() + "/import.txt"
	content := "https://example.com/1 - One\nсломанная строка\nhttps://example.com/2 - Two\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("не удалось создать файл импорта: %v", err)
	}

	stdout, _, code := runCLI(t, "import", file, "--json")
	if code != cli.ExitOK {
		t.Fatalf("import: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	var result struct {
		Added      int `json:"added"`
		Duplicates int `json:"duplicates"`
		Errors     int `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("import: невалидный JSON: %v (%q)", err, stdout)
	}
	if result.Added != 2 || result.Errors != 1 {
		t.Errorf("import: ожидалось added=2 errors=1, получено %+v", result)
	}
}

func TestCLIUsageErrors(t *testing.T) {
	withIsolatedHome(t)

	cases := [][]string{
		{},
		{"нетакойкоманды"},
		{"get"},
		{"get", "abc"},
		{"add", "https://example.com/1"},
		{"update", "1"},
		{"rm"},
		{"export", "-f", "yaml"},
	}
	for _, args := range cases {
		if _, _, code := runCLI(t, args...); code != cli.ExitUsage {
			t.Errorf("%v: ожидался код %d, получен %d", args, cli.ExitUsage, code)
		}
	}
}

func TestCLIGetAndRemoveByURL(t *testing.T) {
	withIsolatedHome(t)

	runCLI(t, "add", "https://example.com/1", "One")
	runCLI(t, "add", "https://example.com/2", "Two")

	stdout, _, code := runCLI(t, "get", "--url", "https://example.com/2", "--json")
	if code != cli.ExitOK {
		t.Fatalf("get --url: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	if !strings.Contains(stdout, "\"title\": \"Two\"") {
		t.Errorf("get --url: неожиданный тикет: %q", stdout)
	}

	if _, _, code := runCLI(t, "get", "--url", "https://example.com/нет"); code != cli.ExitNotFound {
		t.Errorf("get по несуществующей ссылке: ожидался код %d, получен %d", cli.ExitNotFound, code)
	}
	if _, _, code := runCLI(t, "get", "1", "--url", "https://example.com/1"); code != cli.ExitUsage {
		t.Errorf("get с id и --url одновременно: ожидался код %d, получен %d", cli.ExitUsage, code)
	}

	if _, _, code := runCLI(t, "rm", "--url", "https://example.com/1"); code != cli.ExitOK {
		t.Errorf("rm --url: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	stdout, _, _ = runCLI(t, "list")
	if strings.Contains(stdout, "https://example.com/1") {
		t.Errorf("rm --url: тикет не удален: %q", stdout)
	}
}

func TestCLIAddDuplicateReturnsExisting(t *testing.T) {
	withIsolatedHome(t)

	runCLI(t, "add", "https://example.com/1", "One")

	// Дубликат отдает существующий тикет в stdout, но код возврата остается 4
	stdout, _, code := runCLI(t, "add", "https://example.com/1", "Другое название", "--json")
	if code != cli.ExitDuplicate {
		t.Fatalf("add дубликата: ожидался код %d, получен %d", cli.ExitDuplicate, code)
	}
	var existing struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(stdout), &existing); err != nil {
		t.Fatalf("add дубликата: невалидный JSON: %v (%q)", err, stdout)
	}
	if existing.ID != 1 || existing.Title != "One" {
		t.Errorf("add дубликата: ожидался существующий тикет, получено %+v", existing)
	}

	// upsert обновляет название существующего тикета и завершается успешно
	stdout, _, code = runCLI(t, "add", "https://example.com/1", "Обновлено", "--upsert", "--json")
	if code != cli.ExitOK {
		t.Fatalf("add --upsert: ожидался код %d, получен %d", cli.ExitOK, code)
	}
	if !strings.Contains(stdout, "Обновлено") {
		t.Errorf("add --upsert: название не обновилось: %q", stdout)
	}
	stdout, _, _ = runCLI(t, "list")
	if lines := strings.Count(strings.TrimSpace(stdout), "\n") + 1; lines != 1 {
		t.Errorf("add --upsert:новый тикет не должен был создаться, строк: %d", lines)
	}
}

func TestCLIManifest(t *testing.T) {
	withIsolatedHome(t)

	stdout, _, code := runCLI(t, "help", "--json")
	if code != cli.ExitOK {
		t.Fatalf("help --json: ожидался код %d, получен %d", cli.ExitOK, code)
	}

	var m struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		ExitCodes []struct {
			Code int    `json:"code"`
			Name string `json:"name"`
		} `json:"exit_codes"`
		Commands []struct {
			Name    string `json:"name"`
			Aliases []string
			Usage   string `json:"usage"`
			Summary string `json:"summary"`
			Flags   []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"flags"`
		} `json:"commands"`
	}
	if err := json.Unmarshal([]byte(stdout), &m); err != nil {
		t.Fatalf("help --json: невалидный JSON: %v", err)
	}
	if m.Name != "gotickets" || m.Version == "" {
		t.Errorf("манифест: неожиданные name/version: %q %q", m.Name, m.Version)
	}
	if len(m.ExitCodes) != 5 {
		t.Errorf("манифест: ожидалось 5 кодов возврата, получено %d", len(m.ExitCodes))
	}
	for _, c := range m.Commands {
		if c.Usage == "" || c.Summary == "" {
			t.Errorf("манифест: у команды %s пустой usage или summary", c.Name)
		}
	}

	// Каждая команда из манифеста должна диспетчеризоваться и понимать свои флаги
	for _, c := range m.Commands {
		if c.Name == "help" || c.Name == "version" {
			continue
		}
		_, stderr, code := runCLI(t, c.Name, "--json")
		if code == cli.ExitOK || code == cli.ExitUsage || code == cli.ExitNotFound {
			// ожидаемые исходы: команда либо отработала, либо потребовала аргументы
		} else {
			t.Errorf("команда %s: неожиданный код %d (stderr: %q)", c.Name, code, stderr)
		}
		if strings.Contains(stderr, "flag provided but not defined") {
			t.Errorf("команда %s не принимает общий флаг --json", c.Name)
		}
	}
}

func TestCLISpecFlagsAreRegistered(t *testing.T) {
	withIsolatedHome(t)

	// Флаг, объявленный в спецификации, должен существовать в наборе флагов команды:
	// иначе разбор завершится ошибкой "flag provided but not defined"
	cases := []struct {
		args []string
	}{
		{[]string{"list", "-q", "x", "-n", "1"}},
		{[]string{"search", "x", "-n", "1"}},
		{[]string{"get", "--url", "https://example.com/1"}},
		{[]string{"add", "-u", "https://example.com/1", "-t", "T", "--upsert", "--allow-duplicate"}},
		{[]string{"update", "1", "-u", "https://example.com/2", "-t", "T"}},
		{[]string{"rm", "--url", "https://example.com/1"}},
		{[]string{"open", "1", "--print"}},
		{[]string{"export", "-f", "json", "-o", t.TempDir() + "/out.json"}},
		{[]string{"backup", "list"}},
		{[]string{"path"}},
	}
	for _, c := range cases {
		_, stderr, _ := runCLI(t, c.args...)
		if strings.Contains(stderr, "not defined") {
			t.Errorf("%v: флаг из спецификации не зарегистрирован (stderr: %q)", c.args, stderr)
		}
	}
}
