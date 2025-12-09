package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iamruinous/meshtastic-message-relay/internal/relay"
)

// Run starts the TUI with the given relay service
func Run(service *relay.Service) error {
	model := New(service)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}
