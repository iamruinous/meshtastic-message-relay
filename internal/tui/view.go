package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return fmt.Sprintf("%s Initializing...\n", m.spinner.View())
	}

	var b strings.Builder

	// Title
	title := titleStyle.Render("🔗 Meshtastic Message Relay")
	b.WriteString(title)
	b.WriteString("\n")

	// Status bar
	statusBar := m.renderStatusBar()
	b.WriteString(statusBar)
	b.WriteString("\n")

	// Stats
	stats := m.renderStats()
	b.WriteString(stats)
	b.WriteString("\n")

	// Messages viewport
	messagesBox := boxStyle.Width(m.width - 4).Render(m.viewport.View())
	b.WriteString(messagesBox)
	b.WriteString("\n")

	// Error message if any
	if m.errorMessage != "" {
		b.WriteString(errorStyle.Render("Error: " + m.errorMessage))
		b.WriteString("\n")
	}

	// Help
	help := helpStyle.Render("q: quit • c: clear messages • ↑/↓: scroll")
	b.WriteString(help)

	return b.String()
}

func (m Model) renderStatusBar() string {
	// Connection status
	status := StatusIndicator(m.connected)

	// Connection name
	connInfo := ""
	if m.connName != "" {
		connInfo = statLabelStyle.Render(" | ") + statValueStyle.Render(m.connName)
	}

	// Outputs
	outputInfo := statLabelStyle.Render(" | Outputs: ") + statValueStyle.Render(fmt.Sprintf("%d", m.outputCount))

	// Uptime
	uptime := time.Since(m.startTime).Round(time.Second)
	uptimeInfo := statLabelStyle.Render(" | Uptime: ") + statValueStyle.Render(uptime.String())

	return status + connInfo + outputInfo + uptimeInfo
}

func (m Model) renderStats() string {
	received := statLabelStyle.Render("Received: ") + statValueStyle.Render(fmt.Sprintf("%d", m.stats.MessagesReceived))
	sent := statLabelStyle.Render(" | Sent: ") + statValueStyle.Render(fmt.Sprintf("%d", m.stats.MessagesSent))
	filtered := statLabelStyle.Render(" | Filtered: ") + statValueStyle.Render(fmt.Sprintf("%d", m.stats.MessagesFiltered))
	errors := statLabelStyle.Render(" | Errors: ")
	if m.stats.Errors > 0 {
		errors += errorStyle.Render(fmt.Sprintf("%d", m.stats.Errors))
	} else {
		errors += statValueStyle.Render("0")
	}

	return received + sent + filtered + errors
}

func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return statLabelStyle.Render("No messages yet. Waiting for incoming packets...")
	}

	var b strings.Builder
	for _, msg := range m.messages {
		line := m.renderMessage(msg)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderMessage(msg MessageDisplay) string {
	timeStr := messageTimeStyle.Render(msg.Time.Format("15:04:05"))
	from := messageFromStyle.Render(msg.From)
	msgType := messageTypeStyle.Render(fmt.Sprintf("[%s]", msg.Type))

	// Signal info if available
	signalInfo := ""
	if msg.SNR != 0 || msg.RSSI != 0 {
		signalInfo = statLabelStyle.Render(fmt.Sprintf(" (SNR:%.1f RSSI:%d)", msg.SNR, msg.RSSI))
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top, timeStr, " ", from, " ", msgType, signalInfo)

	content := messageContentStyle.Render("  " + msg.Content)

	return header + "\n" + content
}
