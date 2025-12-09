package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// Update handles messages and updates the model
//
//nolint:gocritic // hugeParam: Model must be value receiver to implement tea.Model interface
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "c":
			// Clear messages
			m.messages = make([]MessageDisplay, 0)
			m.viewport.SetContent(m.renderMessages())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 8 // Title + status + stats
		footerHeight := 3 // Help text
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-verticalMargins)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - verticalMargins
		}
		m.viewport.SetContent(m.renderMessages())

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		// Update stats from service
		if m.service != nil {
			m.stats = m.service.GetStats()
			conn := m.service.GetConnection()
			if conn != nil {
				m.connected = conn.IsConnected()
				m.connName = conn.Name()
			}
			m.outputCount = len(m.service.GetOutputs())
		}
		cmds = append(cmds, tickCmd())

	case messageMsg:
		if msg != nil {
			m.addMessage((*message.Packet)(msg))
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		// Continue waiting for messages
		cmds = append(cmds, waitForMessage(m.service))

	case errMsg:
		m.errorMessage = msg.Error()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Handle viewport updates
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) addMessage(msg *message.Packet) {
	fromNode := fmt.Sprintf("!%08x", msg.From)
	if msg.FromNode != nil && msg.FromNode.User != nil {
		if msg.FromNode.User.ShortName != "" {
			fromNode = msg.FromNode.User.ShortName
		}
	}

	var content string
	switch p := msg.Payload.(type) {
	case *message.TextMessage:
		content = p.Text
	case string:
		content = p
	default:
		content = fmt.Sprintf("%v", msg.Payload)
	}

	display := MessageDisplay{
		Time:    msg.ReceivedAt,
		From:    fromNode,
		Type:    msg.PortNum.String(),
		Content: content,
		SNR:     msg.SNR,
		RSSI:    msg.RSSI,
	}

	m.messages = append(m.messages, display)

	// Trim to max messages
	if len(m.messages) > MaxMessages {
		m.messages = m.messages[len(m.messages)-MaxMessages:]
	}
}
