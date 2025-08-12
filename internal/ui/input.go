package ui

import "github.com/charmbracelet/bubbles/textinput"

// createTextInput creates and configures the text input component
func createTextInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Enter text..."
	ti.CharLimit = 500
	ti.Width = 60
	return ti
}

// SetupTextInputForURL configures text input for URL entry
func (m *Model) SetupTextInputForURL() {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter URL..."
	m.textInput.Focus()
	m.tempURL = ""
	m.urlError = ""
}

// SetupTextInputForTitle configures text input for title entry
func (m *Model) SetupTextInputForTitle() {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter ticket title..."
	m.textInput.Focus()
	m.urlError = ""
}

// SetupTextInputForSearch configures text input for search
func (m *Model) SetupTextInputForSearch() {
	m.searchMode = true
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Search tickets..."
	m.textInput.Focus()
}

// SetupTextInputForImport configures text input for import file path
func (m *Model) SetupTextInputForImport() {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter path to .txt file..."
	m.textInput.Focus()
}

// ClearTextInput resets text input to default state
func (m *Model) ClearTextInput() {
	m.textInput.SetValue("")
	m.textInput.Blur()
}