package unit

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"gotickets/pkg/gotickets"
)

func TestNewModel(t *testing.T) {
	model := gotickets.NewModel()

	if model.GetStorage() == nil {
		t.Fatal("Expected storage to be initialized")
	}

	if model.GetViewMode() != gotickets.ViewList {
		t.Fatalf("Expected initial view mode to be ViewList, got %v", model.GetViewMode())
	}
}

func TestModel_Init(t *testing.T) {
	model := gotickets.NewModel()

	cmd := model.Init()
	if cmd == nil {
		t.Fatal("Expected Init() to return a command")
	}
}

func TestModel_Update_WindowResize(t *testing.T) {
	model := gotickets.NewModel()

	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := model.Update(resizeMsg)

	// Basic check that the model was updated
	if updatedModel == nil {
		t.Fatal("Expected Update() to return a model")
	}
}
