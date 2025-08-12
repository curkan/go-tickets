package gotickets

import (
	"gotickets/internal/storage"
)

// Ticket type alias for backward compatibility
type Ticket = storage.Ticket

// TicketStorage type alias for backward compatibility
type TicketStorage = storage.TicketStorage

// ImportResult type alias for backward compatibility
type ImportResult = storage.ImportResult

// FileSystem type alias for backward compatibility
type FileSystem = storage.FileSystem

// RealFileSystem type alias for backward compatibility
type RealFileSystem = storage.RealFileSystem

// Функции-обертки для обратной совместимости

// NewTicketStorage создает новое хранилище тикетов
func NewTicketStorage(fs storage.FileSystem) *TicketStorage {
	return storage.NewTicketStorage(fs)
}

// LoadTickets загружает тикеты из стандартного места
func LoadTickets() (*TicketStorage, error) {
	return storage.LoadTicketsWithFS(&storage.RealFileSystem{})
}

// LoadTicketsWithFS загружает тикеты с использованием предоставленной файловой системы
func LoadTicketsWithFS(fs storage.FileSystem) (*TicketStorage, error) {
	return storage.LoadTicketsWithFS(fs)
}

// LoadTicketsFromPath загружает тикеты из конкретного файла
func LoadTicketsFromPath(filePath string) (*TicketStorage, error) {
	return storage.LoadTicketsFromPathWithFS(&storage.RealFileSystem{}, filePath)
}

// LoadTicketsFromPathWithFS загружает тикеты из конкретного файла с использованием предоставленной ФС
func LoadTicketsFromPathWithFS(fs storage.FileSystem, filePath string) (*TicketStorage, error) {
	return storage.LoadTicketsFromPathWithFS(fs, filePath)
}

// listBackups возвращает список доступных резервных копий
func listBackups() ([]string, error) {
	return storage.ListBackupsUsing(&storage.RealFileSystem{})
}

// listBackupsUsing возвращает список резервных копий с использованием предоставленной ФС
func listBackupsUsing(fs storage.FileSystem) ([]string, error) {
	return storage.ListBackupsUsing(fs)
}

// restoreFromBackup восстанавливает данные из резервной копии
func restoreFromBackup(backupName string) error {
	return storage.RestoreFromBackupUsing(&storage.RealFileSystem{}, backupName)
}

// restoreFromBackupUsing восстанавливает данные из резервной копии с использованием предоставленной ФС
func restoreFromBackupUsing(fs storage.FileSystem, backupName string) error {
	return storage.RestoreFromBackupUsing(fs, backupName)
}
