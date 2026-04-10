// Package tui implements the Bubbletea TUI for the jules-installer.
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/Jules-Solutions/jules-installer/internal/auth"
)

// state represents which screen the installer is currently showing.
type state int

const (
	stateWelcome  state = iota // splash/intro screen
	stateAuth                  // authentication flow
	stateAudit                 // environment audit
	stateSetup                 // interactive questions
	stateDownload              // vault download
	stateConfig                // config writing
	stateDone                  // completion / handoff
	stateError                 // fatal error
)

// authState tracks sub-state within the auth screen.
type authState int

const (
	authStateBrowser authState = iota // waiting for browser callback
	authStateDevice                   // waiting for device code auth
	authStateAPIKey                   // user is pasting an API key
	authStateSuccess                  // auth complete, ready to continue
)

// --- Messages ---

// tickMsg drives spinner animation.
type tickMsg time.Time

// authDoneMsg is sent when any auth flow completes.
type authDoneMsg struct {
	apiKey string
	method auth.Method
	err    error
}

// deviceCodeMsg is sent once we have a device code to display.
type deviceCodeMsg struct {
	userCode        string
	verificationURI string
}

// --- Model ---

// Model is the root Bubbletea model for the installer.
type Model struct {
	// Current screen.
	state state

	// Terminal dimensions (updated on tea.WindowSizeMsg).
	width  int
	height int

	// Auth sub-state.
	authState      authState
	authErrMsg     string
	deviceCode     string
	deviceVerifyURI string
	textInput      *textinput.Model

	// Auth result.
	apiKey    string
	authMethod auth.Method

	// Auth service base URL.
	authURL string

	// Audit results (populated after stateAudit).
	auditResults []interface{} // placeholder; will be []audit.Check

	// Spinner animation frame counter.
	spinnerFrame int

	// Build version string shown on welcome screen.
	version string

	// Non-nil when the installer hits a fatal error.
	err error
}

// NewModel creates a fresh installer Model with the given auth URL and version string.
func NewModel(authURL, version string) Model {
	ti := textinput.New()
	ti.Placeholder = "dck_..."
	ti.CharLimit = 128
	ti.Width = 60

	return Model{
		state:   stateWelcome,
		authURL: authURL,
		version: version,
		textInput: &ti,
	}
}

// --- Bubbletea interface ---

// Init is called once before the first Update. Starts the spinner tick.
func (m Model) Init() tea.Cmd {
	return tickCmd()
}

// Update handles all incoming messages and keyboard events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.spinnerFrame++
		return m, tickCmd()

	case authDoneMsg:
		if msg.err != nil {
			// Fall through to next auth method or show error.
			return m.handleAuthError(msg.err)
		}
		m.apiKey = msg.apiKey
		m.authMethod = msg.method
		m.authState = authStateSuccess
		return m, nil

	case deviceCodeMsg:
		m.deviceCode = msg.userCode
		m.deviceVerifyURI = msg.verificationURI
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate text input events when on API key screen.
	if m.state == stateAuth && m.authState == authStateAPIKey && m.textInput != nil {
		ti, cmd := m.textInput.Update(msg)
		m.textInput = &ti
		return m, cmd
	}

	return m, nil
}

// View renders the current screen as a string.
func (m Model) View() string {
	switch m.state {
	case stateWelcome:
		return renderWelcome(m)
	case stateAuth:
		return renderAuth(m)
	case stateAudit:
		return renderAudit(m)
	case stateSetup:
		return renderSetup(m)
	case stateDownload:
		return renderDownload(m)
	case stateConfig:
		return renderConfig(m)
	case stateDone:
		return renderDone(m)
	case stateError:
		return renderError(m)
	default:
		return ""
	}
}

// --- Key handling ---

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

	case stateWelcome:
		switch msg.String() {
		case "enter", " ":
			return m.startAuth()
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case stateAuth:
		switch m.authState {

		case authStateBrowser:
			switch msg.String() {
			case "d":
				return m.switchToDeviceFlow()
			case "k":
				m.authState = authStateAPIKey
				if m.textInput != nil {
					m.textInput.Focus()
				}
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case authStateDevice:
			switch msg.String() {
			case "k":
				m.authState = authStateAPIKey
				if m.textInput != nil {
					m.textInput.Focus()
				}
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case authStateAPIKey:
			switch msg.String() {
			case "enter":
				return m.submitAPIKey()
			case "esc":
				m.authState = authStateBrowser
				m.authErrMsg = ""
				return m.startBrowserFlow()
			case "ctrl+c":
				return m, tea.Quit
			}
			// Other keys are handled by the text input component above.

		case authStateSuccess:
			switch msg.String() {
			case "enter", " ":
				// Advance to audit (stub — goes directly to done for now).
				m.state = stateDone
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}

	case stateAudit, stateSetup, stateDownload, stateConfig:
		// Stub states just pass through on Enter.
		switch msg.String() {
		case "enter", " ":
			m.state++
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case stateDone, stateError:
		switch msg.String() {
		case "q", "ctrl+c", "enter":
			return m, tea.Quit
		}
	}

	return m, nil
}

// --- Auth flow helpers ---

// startAuth transitions to the auth screen and begins the browser flow.
func (m Model) startAuth() (tea.Model, tea.Cmd) {
	m.state = stateAuth
	m.authState = authStateBrowser
	return m, tea.Batch(tickCmd(), m.runBrowserFlowCmd())
}

// startBrowserFlow re-launches the browser flow (e.g. after returning from API key screen).
func (m Model) startBrowserFlow() (tea.Model, tea.Cmd) {
	return m, m.runBrowserFlowCmd()
}

// switchToDeviceFlow starts the device code flow.
func (m Model) switchToDeviceFlow() (tea.Model, tea.Cmd) {
	m.authState = authStateDevice
	m.deviceCode = ""
	m.deviceVerifyURI = ""
	return m, m.runDeviceFlowCmd()
}

// submitAPIKey reads the text input and validates the pasted key.
func (m Model) submitAPIKey() (tea.Model, tea.Cmd) {
	if m.textInput == nil {
		return m, nil
	}
	key := m.textInput.Value()
	return m, m.runAPIKeyFlowCmd(key)
}

func (m Model) handleAuthError(err error) (tea.Model, tea.Cmd) {
	// On browser failure, prompt the user to choose another method.
	if m.authState == authStateBrowser {
		m.authErrMsg = fmt.Sprintf("Browser auth failed: %v", err)
		// Stay on browser screen; user can press d or k.
		return m, nil
	}
	m.err = err
	m.state = stateError
	return m, nil
}

// --- Async command runners ---

// runBrowserFlowCmd runs the browser callback flow in a goroutine and returns
// the result as an authDoneMsg.
func (m Model) runBrowserFlowCmd() tea.Cmd {
	authURL := m.authURL
	return func() tea.Msg {
		key, err := auth.BrowserFlowPublic(authURL)
		return authDoneMsg{apiKey: key, method: auth.MethodBrowser, err: err}
	}
}

// runDeviceFlowCmd runs the device code flow in a goroutine.
// It sends a deviceCodeMsg when the user code is ready, then an authDoneMsg
// on completion.
func (m Model) runDeviceFlowCmd() tea.Cmd {
	authURL := m.authURL
	// We use a channel to bridge the progress callback into Bubbletea's message bus.
	// However, Bubbletea v2 Cmd must return exactly one Msg. For device code we
	// return the final result and rely on the polling progress being reflected
	// via a separate tick-driven approach in a future iteration.
	//
	// For Phase 1, we run the full device flow synchronously in the Cmd and
	// return the final authDoneMsg. Progress updates (deviceCodeMsg) are not
	// streamed in this iteration.
	return func() tea.Msg {
		key, err := auth.DeviceFlowPublic(authURL)
		return authDoneMsg{apiKey: key, method: auth.MethodDevice, err: err}
	}
}

// runAPIKeyFlowCmd validates the user-pasted API key.
func (m Model) runAPIKeyFlowCmd(key string) tea.Cmd {
	authURL := m.authURL
	return func() tea.Msg {
		validated, err := auth.APIKeyFlowPublic(authURL, key)
		return authDoneMsg{apiKey: validated, method: auth.MethodAPIKey, err: err}
	}
}

// tickCmd returns a command that fires a tickMsg after a short interval,
// driving the spinner animation.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
