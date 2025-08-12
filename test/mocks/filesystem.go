package mocks

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MockFileSystem implements storage.FileSystem for testing
type MockFileSystem struct {
	homeDir     string
	files       map[string][]byte
	directories map[string]bool
	statResults map[string]os.FileInfo
	errors      map[string]error
}

func NewMockFileSystem(homeDir string) *MockFileSystem {
	return &MockFileSystem{
		homeDir:     homeDir,
		files:       make(map[string][]byte),
		directories: make(map[string]bool),
		statResults: make(map[string]os.FileInfo),
		errors:      make(map[string]error),
	}
}

func (fs *MockFileSystem) UserHomeDir() (string, error) {
	if err, exists := fs.errors["UserHomeDir"]; exists {
		return "", err
	}
	return fs.homeDir, nil
}

func (fs *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err, exists := fs.errors["MkdirAll"]; exists {
		return err
	}
	fs.directories[path] = true
	return nil
}

func (fs *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if err, exists := fs.errors["ReadFile"]; exists {
		return nil, err
	}
	if data, exists := fs.files[filename]; exists {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (fs *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err, exists := fs.errors["WriteFile"]; exists {
		return err
	}
	fs.files[filename] = data
	return nil
}

func (fs *MockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	if err, exists := fs.errors["ReadDir"]; exists {
		return nil, err
	}
	
	var entries []os.DirEntry
	for filename := range fs.files {
		if strings.HasPrefix(filename, dirname+"/") || strings.HasPrefix(filename, dirname+"\\") {
			relPath := strings.TrimPrefix(filename, dirname+"/")
			relPath = strings.TrimPrefix(relPath, dirname+"\\")
			if !strings.Contains(relPath, "/") && !strings.Contains(relPath, "\\") {
				entries = append(entries, &mockDirEntry{name: relPath, isDir: false})
			}
		}
	}
	
	return entries, nil
}

func (fs *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if err, exists := fs.errors["Stat"]; exists {
		return nil, err
	}
	if info, exists := fs.statResults[name]; exists {
		return info, nil
	}
	if _, exists := fs.files[name]; exists {
		return &mockFileInfo{name: filepath.Base(name), size: int64(len(fs.files[name]))}, nil
	}
	return nil, os.ErrNotExist
}

func (fs *MockFileSystem) Open(name string) (*os.File, error) {
	if err, exists := fs.errors["Open"]; exists {
		return nil, err
	}
	if _, exists := fs.files[name]; exists {
		// For testing purposes, create a temporary file with the content
		tmpfile, err := os.CreateTemp("", "mock")
		if err != nil {
			return nil, err
		}
		if _, err := tmpfile.Write(fs.files[name]); err != nil {
			tmpfile.Close()
			return nil, err
		}
		if _, err := tmpfile.Seek(0, 0); err != nil {
			tmpfile.Close()
			return nil, err
		}
		return tmpfile, nil
	}
	return nil, os.ErrNotExist
}

// SetError sets an error for a specific method
func (fs *MockFileSystem) SetError(method string, err error) {
	fs.errors[method] = err
}

type mockFileInfo struct {
	name string
	size int64
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode          { return 0644 }
func (m *mockDirEntry) Info() (os.FileInfo, error) { 
	return &mockFileInfo{name: m.name, size: 0}, nil 
}

// AssertErr is a helper to produce deterministic errors
type AssertErr string

func (e AssertErr) Error() string { return string(e) }
