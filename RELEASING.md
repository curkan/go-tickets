# Руководство по релизам с GoReleaser

## Что такое GoReleaser?

GoReleaser - это инструмент для автоматизации процесса создания релизов Go приложений. Он автоматически собирает бинарные файлы для разных платформ, создает архивы, генерирует changelog и публикует релизы на GitHub.

## Установка GoReleaser

### macOS (Homebrew)
```bash
brew install goreleaser/tap/goreleaser
```

### Linux
```bash
# Скачать последнюю версию
curl -sfL https://goreleaser.com/static/run | bash
```

### Windows
```bash
# Скачать последнюю версию
curl -sfL https://goreleaser.com/static/run | bash
```

## Настройка проекта

1. **Создан файл `.goreleaser.yml`** с конфигурацией для проекта `gotickets`
2. **Настроена публикация в Homebrew** через репозиторий `curkan/homebrew-public`

## Как использовать

### 1. Подготовка к релизу

Убедитесь, что все изменения закоммичены и запушены в репозиторий:

```bash
git add .
git commit -m "feat: add new feature"
git push origin main
```

### 2. Создание тега версии

Создайте тег для новой версии:

```bash
# Для patch версии (1.0.0 -> 1.0.1)
git tag -a v1.0.1 -m "Release v1.0.1"

# Для minor версии (1.0.0 -> 1.1.0)
git tag -a v1.1.0 -m "Release v1.1.0"

# Для major версии (1.0.0 -> 2.0.0)
git tag -a v2.0.0 -m "Release v2.0.0"
```

### 3. Запуск релиза

#### Локальный тест (без публикации)
```bash
goreleaser release --snapshot --rm-dist
```

#### Полный релиз с публикацией
```bash
# Сначала запушьте тег
git push origin v1.0.1

# Затем запустите релиз
goreleaser release
```

### 4. Что происходит при релизе

1. **Сборка**: GoReleaser собирает бинарные файлы для:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64, arm64)

2. **Архивирование**: Создаются архивы:
   - `gotickets_Linux_x86_64.tar.gz`
   - `gotickets_Darwin_x86_64.tar.gz`
   - `gotickets_Windows_x86_64.zip`

3. **Changelog**: Автоматически генерируется changelog на основе коммитов

4. **GitHub Release**: Создается релиз на GitHub с:
   - Описанием изменений
   - Скачиваемыми архивами
   - Чексуммами

5. **Homebrew**: Обновляется формула в `curkan/homebrew-public`

## Конфигурация

### Основные настройки в `.goreleaser.yml`:

- **`brews`**: Настройки для публикации в Homebrew
  - `name: gotickets` - название формулы
  - `homepage`: ссылка на репозиторий
  - `repository`: репозиторий для Homebrew

- **`builds`**: Настройки сборки
  - `CGO_ENABLED=0` - статическая сборка
  - Поддержка Linux, Windows, macOS

- **`archives`**: Настройки архивирования
  - Формат имен файлов
  - ZIP для Windows, tar.gz для остальных

## Установка через Homebrew

После релиза пользователи смогут установить приложение:

```bash
# Добавить tap
brew tap curkan/homebrew-public

# Установить gotickets
brew install gotickets
```

## Troubleshooting

### Проблемы с правами доступа
Убедитесь, что у вас есть права на:
- Создание релизов в GitHub репозитории
- Пуш в `curkan/homebrew-public`

### Проблемы с токенами
Настройте GitHub токен:
```bash
export GITHUB_TOKEN=your_token_here
```

### Проверка конфигурации
```bash
goreleaser check
```

## Полезные команды

```bash
# Проверить конфигурацию
goreleaser check

# Тестовый релиз
goreleaser release --snapshot --rm-dist

# Показать help
goreleaser --help

# Информация о версии
goreleaser version
```

## Ссылки

- [Официальная документация GoReleaser](https://goreleaser.com/)
- [GitHub репозиторий](https://github.com/goreleaser/goreleaser)
- [Homebrew tap репозиторий](https://github.com/curkan/homebrew-public)
