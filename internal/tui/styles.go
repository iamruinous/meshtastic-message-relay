package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	errorColor     = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1).
			MarginBottom(1)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)

	// Status styles
	connectedStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	disconnectedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	// Spinner style
	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// Stats styles
	statLabelStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	statValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	// Message styles
	messageTimeStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	messageFromStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	messageTypeStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	messageContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(1, 0)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)
)

// StatusIndicator returns a styled status indicator
func StatusIndicator(connected bool) string {
	if connected {
		return connectedStyle.Render("● Connected")
	}
	return disconnectedStyle.Render("○ Disconnected")
}
