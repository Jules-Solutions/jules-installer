// Package tui implements the Bubbletea TUI for the jules-installer.
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/Jules-Solutions/jules-installer/internal/audit"
	"github.com/Jules-Solutions/jules-installer/internal/auth"
	"github.com/Jules-Solutions/jules-installer/internal/config"
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

// setupState tracks sub-state within the setup screen.
type setupState int

const (
	setupVaultPath  setupState = iota // user selects vault directory
	setupConfirmMCP                   // confirm MCP configuration
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

// auditDoneMsg is sent when the environment audit completes.
type auditDoneMsg struct {
	checks []audit.Check
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
	auditResults []audit.Check

	// Setup sub-state and inputs.
	setupState      setupState
	setupVaultInput *textinput.Model
	setupConfigMCP  bool

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

	// Vault path input with sensible default.
	vi := textinput.New()
	vi.SetValue(config.DefaultVaultPath())
	vi.Placeholder = "~/{Name}.Life"
	vi.CharLimit = 256
	vi.Width = 60

	return Model{
		state:           stateWelcome,
		authURL:         authURL,
		version:         version,
		textInput:       &ti,
		setupVaultInput: &vi,
		setupConfigMCP:  true, // default yes
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

	case auditDoneMsg:
		m.auditResults = msg.checks
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

	// Delegate text input events when on setup vault path screen.
	if m.state == stateSetup && m.setupState == setupVaultPath && m.setupVaultInput != nil {
		vi, cmd := m.setupVaultInput.Update(msg)
		m.setupVaultInput = &vi
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
				// Save API key to config before advancing.
				cfg := config.DefaultConfig()
				cfg.Auth.APIKey = m.apiKey
				cfg.Auth.AuthURL = m.authURL
				if err := config.SaveConfig(cfg); err != nil {
					m.err = fmt.Errorf("saving config: %w", err)
					m.state = stateError
					return m, nil
				}
				// Start the environment audit.
				m.state = stateAudit
				return m, m.runAuditCmd()
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}

	case stateAudit:
		// Wait for audit results, then Enter to continue.
		if len(m.auditResults) > 0 {
			switch msg.String() {
			case "enter", " ":
				m.state = stateSetup
				m.setupState = setupVaultPath
				if m.setupVaultInput != nil {
					m.setupVaultInput.Focus()
				}
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		} else {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}

	case stateSetup:
		switch m.setupState {
		case setupVaultPath:
			switch msg.String() {
			case "enter":
				// Vault path confirmed, move to MCP question.
				m.setupState = setupConfirmMCP
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			// Other keys handled by textinput delegate above.

		case setupConfirmMCP:
			switch msg.String() {
			case "left", "right", "h", "l", "tab":
				m.setupConfigMCP = !m.setupConfigMCP
				return m, nil
			case "y":
				m.setupConfigMCP = true
				return m, nil
			case "n":
				m.setupConfigMCP = false
				return m, nil
			case "enter":
				// Save full config and advance to done.
				return m.finishSetup()
			case "ctrl+c":
				return m, tea.Quit
			}
		}

	case stateDownload, stateConfig:
		// Still stubs — skip through.
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

// runAuditCmd runs all environment checks in a background goroutine.
func (m Model) runAuditCmd() tea.Cmd {
	return func() tea.Msg {
		return auditDoneMsg{checks: audit.RunAudit()}
	}
}

// finishSetup saves the final config from all collected answers and advances to Done.
func (m Model) finishSetup() (tea.Model, tea.Cmd) {
	vaultPath := ""
	if m.setupVaultInput != nil {
		vaultPath = m.setupVaultInput.Value()
	}

	cfg, _ := config.LoadConfig()
	cfg.Auth.APIKey = m.apiKey
	cfg.Auth.AuthURL = m.authURL
	cfg.Auth.APIURL = "https://api.jules.solutions"
	cfg.Local.VaultPath = vaultPath
	if err := config.SaveConfig(cfg); err != nil {
		m.err = fmt.Errorf("saving config: %w", err)
		m.state = stateError
		return m, nil
	}

	m.state = stateDone
	return m, nil
}

// tickCmd returns a command that fires a tickMsg after a short interval,
// driving the spinner animation.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
