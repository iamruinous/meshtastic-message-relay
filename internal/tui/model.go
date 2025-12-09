// Package tui provides the terminal user interface.
package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iamruinous/meshtastic-message-relay/internal/message"
	"github.com/iamruinous/meshtastic-message-relay/internal/relay"
)

// MaxMessages is the maximum number of messages to display
const MaxMessages = 100

// Model represents the TUI state
type Model struct {
	// Service reference
	service *relay.Service

	// UI state
	width    int
	height   int
	ready    bool
	quitting bool

	// Components
	spinner  spinner.Model
	viewport viewport.Model

	// Data
	messages     []MessageDisplay
	connected    bool
	connName     string
	outputCount  int
	stats        relay.Stats
	startTime    time.Time
	lastUpdate   time.Time
	errorMessage string
}

// MessageDisplay holds a message for display
type MessageDisplay struct {
	Time    time.Time
	From    string
	Type    string
	Content string
	SNR     float32
	RSSI    int32
}

// New creates a new TUI model
func New(service *relay.Service) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		service:   service,
		spinner:   s,
		messages:  make([]MessageDisplay, 0),
		startTime: time.Now(),
	}
}

// Init initializes the model
//
//nolint:gocritic // hugeParam: Model must be value receiver to implement tea.Model interface
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		waitForMessage(m.service),
	)
}

// tickMsg is sent periodically to update the UI
type tickMsg time.Time

// messageMsg is sent when a new message arrives
type messageMsg *message.Packet

// errMsg is sent when an error occurs
type errMsg error

// tickCmd returns a command that sends a tick every second
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// waitForMessage waits for messages from the service
func waitForMessage(svc *relay.Service) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return nil
		}
		conn := svc.GetConnection()
		if conn == nil {
			return nil
		}
		msg, ok := <-conn.Messages()
		if !ok {
			return nil
		}
		return messageMsg(msg)
	}
}
