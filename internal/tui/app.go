// Package tui implements the Bubbletea TUI for the jules-installer.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/Jules-Solutions/jules-installer/internal/audit"
	"github.com/Jules-Solutions/jules-installer/internal/auth"
	"github.com/Jules-Solutions/jules-installer/internal/config"
	"github.com/Jules-Solutions/jules-installer/internal/setup"
)

// state represents which screen the installer is currently showing.
type state int

const (
	stateWelcome  state = iota // splash/intro screen
	stateTier                  // Tier 1 vs Tier 2 picker (after Welcome, before Auth)
	stateRerun                 // Re-run menu shown when a valid config.toml is already present
	stateAuth                  // authentication flow
	stateAudit                 // environment audit
	stateSetup                 // interactive questions (Tier 1 only)
	stateDownload              // vault download (Tier 1 only)
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

// tierChoice tracks which tier option is currently highlighted on the tier screen.
type tierChoice int

const (
	tierChoiceFull   tierChoice = iota // tier1 — full local install
	tierChoiceRemote                   // tier2 — remote MCP-only
)

// rerunChoice tracks which re-run action is currently highlighted.
type rerunChoice int

const (
	rerunChangeTier rerunChoice = iota // switch tier and re-run setup
	rerunReAudit                       // re-run environment audit only
	rerunRewriteMCP                    // re-write .mcp.json with current config
	rerunExit                          // exit without changes
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

// installDoneMsg is sent when auto-install completes.
type installDoneMsg struct {
	results []audit.InstallResult
}

// vaultDownloadMsg is sent when vault download/scaffold completes.
type vaultDownloadMsg struct {
	method string // "git_clone", "scaffold", "existing"
	err    error
}

// launchDoneMsg is sent after the Claude Code launch attempt completes.
type launchDoneMsg struct {
	err error
}

// auditSubState tracks sub-state within the audit screen.
type auditSubState int

const (
	auditShowResults  auditSubState = iota // showing check results
	auditOfferInstall                      // asking "install missing?"
	auditInstalling                        // running installs
	auditRecheck                           // re-auditing after install
)

// --- Model ---

// Model is the root Bubbletea model for the installer.
type Model struct {
	// Current screen.
	state state

	// Terminal dimensions (updated on tea.WindowSizeMsg).
	width  int
	height int

	// Tier — the chosen onboarding path. Empty until the user picks on the tier
	// screen or passes --tier on the command line.
	tier       config.Tier
	tierCursor tierChoice

	// Re-run menu cursor position.
	rerunCursor      rerunChoice
	rerunMessage     string // transient status message after a re-run action
	rerunMessageErr  bool

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

	// Audit results and install state.
	auditResults   []audit.Check
	auditSubState  auditSubState
	installResults []audit.InstallResult

	// Setup sub-state and inputs.
	setupState      setupState
	setupVaultInput *textinput.Model
	setupConfigMCP  bool

	// Vault download result.
	vaultDownloadMethod string // "git_clone", "scaffold", "existing", ""
	vaultDownloadErr    error

	// Path where .mcp.json was actually written (populated after writeConfigAndFinish).
	// Tier 1 → vault_path/.mcp.json; Tier 2 → ~/.claude/.mcp.json.
	mcpPathWritten string

	// Spinner animation frame counter.
	spinnerFrame int

	// Build version string shown on welcome screen.
	version string

	// Non-nil when the installer hits a fatal error.
	err error

	// jules-local install result.
	installLocalErr error

	// Claude Code launch state (Done screen).
	launchAttempted bool
	launchErr       error

	// resume flag: if true, skip completed steps on startup.
	resume bool
}

// ModelOptions bundles the configuration passed from main.go into NewModelWithOptions.
// This exists so we can extend the constructor without breaking callers that only
// set some fields — zero values behave identically to the pre-tier installer.
type ModelOptions struct {
	// AuthURL is the base URL of the auth service (e.g. https://auth.jules.solutions).
	AuthURL string
	// Version is the build version string shown on the welcome screen.
	Version string
	// Resume, when true, skips completed steps on startup per the v0.2.0 --resume flag.
	Resume bool
	// Tier, when non-empty, bypasses the interactive tier picker. Accepts
	// config.TierFull or config.TierRemote. Empty means "prompt the user."
	Tier config.Tier
}

// NewModel creates a fresh installer Model with the given auth URL and version string.
// Kept for backward compatibility with any external callers; main.go uses
// NewModelWithOptions so --tier and other flags can be threaded in.
func NewModel(authURL, version string) Model {
	return NewModelWithOptions(ModelOptions{AuthURL: authURL, Version: version})
}

// NewModelWithResume creates a Model that will skip completed steps on Init.
// Kept for backward compatibility; prefer NewModelWithOptions.
func NewModelWithResume(authURL, version string) Model {
	return NewModelWithOptions(ModelOptions{AuthURL: authURL, Version: version, Resume: true})
}

// NewModelWithOptions is the canonical constructor. Unset fields fall back to
// the same defaults the prior constructors used.
func NewModelWithOptions(opts ModelOptions) Model {
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
		authURL:         opts.AuthURL,
		version:         opts.Version,
		textInput:       &ti,
		setupVaultInput: &vi,
		setupConfigMCP:  true, // default yes
		resume:          opts.Resume,
		tier:            opts.Tier,
	}
}

// --- Bubbletea interface ---

// Init is called once before the first Update. Starts the spinner tick.
// When m.resume is true, scans existing config and skips completed steps.
func (m Model) Init() tea.Cmd {
	if m.resume {
		return tea.Batch(tickCmd(), m.resumeCmd())
	}
	return tickCmd()
}

// resumeCmd checks existing state and returns a command that advances to the
// first incomplete step.
func (m Model) resumeCmd() tea.Cmd {
	return func() tea.Msg {
		return resumeDetectedMsg(detectResumeState())
	}
}

// resumeState encodes which step to jump to when resuming.
type resumeState int

const (
	resumeFromWelcome  resumeState = iota // nothing skipped
	resumeFromAudit                       // auth done, skip to audit
	resumeFromSetup                       // audit done, skip to setup
	resumeFromDownload                    // setup done, skip to download
	resumeFromDone                        // everything done, skip to done
)

// resumeDetectedMsg is sent on startup when --resume is active.
type resumeDetectedMsg resumeState

// detectResumeState reads config.toml and the vault to figure out how far
// along the install is, returning the resumeState to jump to.
func detectResumeState() resumeState {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.Auth.APIKey == "" || !strings.HasPrefix(cfg.Auth.APIKey, "dck_") {
		return resumeFromWelcome
	}

	// Auth is done. Check vault.
	vaultPath := cfg.Local.VaultPath
	if vaultPath == "" {
		return resumeFromAudit
	}

	// Check whether vault has content.
	info, err := os.Stat(vaultPath)
	if err != nil || !info.IsDir() {
		return resumeFromSetup
	}
	entries, _ := os.ReadDir(vaultPath)
	if len(entries) == 0 {
		return resumeFromSetup
	}

	// Vault exists — check for .mcp.json.
	mcpPath := filepath.Join(vaultPath, ".mcp.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		return resumeFromDownload
	}

	return resumeFromDone
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

	case resumeDetectedMsg:
		return m.applyResume(resumeState(msg))

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
		// If tier-relevant installable items exist, offer to install.
		// Tier 2 users never get offered Python/Docker/Node.
		if audit.CountInstallableForTier(msg.checks, string(m.tier)) > 0 {
			m.auditSubState = auditOfferInstall
		} else {
			m.auditSubState = auditShowResults
		}
		return m, nil

	case installDoneMsg:
		m.installResults = msg.results
		// Re-audit after install to show updated results.
		m.auditSubState = auditRecheck
		return m, m.runAuditCmd()

	case vaultDownloadMsg:
		m.vaultDownloadMethod = msg.method
		m.vaultDownloadErr = msg.err
		// Auto-advance to config writing, then done.
		return m.writeConfigAndFinish()

	case launchDoneMsg:
		m.launchAttempted = true
		m.launchErr = msg.err
		// If launch succeeded, quit the installer (Claude Code is opening).
		// If it failed, stay on Done screen so user can see the manual instructions.
		if msg.err == nil {
			return m, tea.Quit
		}
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
	case stateTier:
		return renderTier(m)
	case stateRerun:
		return renderRerun(m)
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
			return m.postWelcome()
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case stateTier:
		switch msg.String() {
		case "up", "k", "left", "h":
			if m.tierCursor > tierChoiceFull {
				m.tierCursor--
			}
			return m, nil
		case "down", "j", "right", "l":
			if m.tierCursor < tierChoiceRemote {
				m.tierCursor++
			}
			return m, nil
		case "1":
			m.tierCursor = tierChoiceFull
			return m, nil
		case "2":
			m.tierCursor = tierChoiceRemote
			return m, nil
		case "tab":
			// Tab toggles between the two choices for quick switching.
			if m.tierCursor == tierChoiceFull {
				m.tierCursor = tierChoiceRemote
			} else {
				m.tierCursor = tierChoiceFull
			}
			return m, nil
		case "enter", " ":
			return m.confirmTier()
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case stateRerun:
		switch msg.String() {
		case "up", "k":
			if m.rerunCursor > rerunChangeTier {
				m.rerunCursor--
			}
			return m, nil
		case "down", "j":
			if m.rerunCursor < rerunExit {
				m.rerunCursor++
			}
			return m, nil
		case "1":
			m.rerunCursor = rerunChangeTier
			return m, nil
		case "2":
			m.rerunCursor = rerunReAudit
			return m, nil
		case "3":
			m.rerunCursor = rerunRewriteMCP
			return m, nil
		case "4":
			m.rerunCursor = rerunExit
			return m, nil
		case "enter", " ":
			return m.confirmRerun()
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
				// Save API key + tier to config before advancing.
				// Merge with existing config so we don't clobber a user's hand-edited
				// vault_path / mcp_path on a re-run.
				cfg, _ := config.LoadConfig()
				if cfg.Auth.APIURL == "" {
					cfg.Auth.APIURL = "https://api.jules.solutions"
				}
				if cfg.Auth.MCPURL == "" {
					cfg.Auth.MCPURL = "https://mcp.jules.solutions/sse"
				}
				cfg.Auth.APIKey = m.apiKey
				cfg.Auth.AuthURL = m.authURL
				cfg.Local.Tier = m.tier
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
		switch m.auditSubState {
		case auditOfferInstall:
			switch msg.String() {
			case "y", "Y":
				// User wants to install missing tools.
				m.auditSubState = auditInstalling
				return m, m.runInstallCmd()
			case "n", "N", "enter", " ":
				// Skip install, continue to setup.
				m.auditSubState = auditShowResults
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}

		case auditShowResults, auditRecheck:
			if len(m.auditResults) > 0 {
				switch msg.String() {
				case "enter", " ":
					// Tier 2 has no vault → skip Setup (vault path + MCP y/n question)
					// and go directly to writing the user-global .mcp.json.
					if m.tier == config.TierRemote {
						return m.finishTier2()
					}
					m.state = stateSetup
					m.setupState = setupVaultPath
					if m.setupVaultInput != nil {
						m.setupVaultInput.Focus()
					}
					return m, nil
				case "q", "ctrl+c":
					return m, tea.Quit
				}
			} else if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

		case auditInstalling:
			// Busy installing — only allow quit.
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

	case stateDone:
		switch msg.String() {
		case "enter", " ":
			// Tier 2 has no vault — there's nothing to "cd and launch" into.
			// Just exit the installer; the user restarts CC themselves.
			if m.tier == config.TierRemote {
				return m, tea.Quit
			}
			// Tier 1: attempt to launch Claude Code in the vault.
			if m.launchAttempted {
				return m, tea.Quit
			}
			return m, m.runLaunchCmd()
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case stateError:
		switch msg.String() {
		case "q", "ctrl+c", "enter":
			return m, tea.Quit
		}
	}

	return m, nil
}

// --- Welcome / Tier / Re-run transition helpers ---

// postWelcome decides what happens after the user presses Enter on the welcome screen.
// Priority order:
//   1. --resume flag was passed: run the resume detection from v0.2.0 unchanged.
//   2. Valid config already exists AND tier is recorded: show the re-run menu.
//   3. Tier was supplied via --tier flag: skip picker, go straight to auth.
//   4. No tier set yet: show the tier picker.
func (m Model) postWelcome() (tea.Model, tea.Cmd) {
	if m.resume {
		// --resume keeps v0.2.0 linear-skip behaviour (resumeDetectedMsg already
		// fired in Init). Drop through to Auth which has its own config check.
		return m.startAuth()
	}

	cfg, err := config.LoadConfig()
	validKey := err == nil && strings.HasPrefix(cfg.Auth.APIKey, "dck_")

	// Re-run menu: valid config + tier already chosen + we're NOT overriding tier via flag.
	if validKey && cfg.Local.Tier.Valid() && m.tier == "" {
		m.apiKey = cfg.Auth.APIKey
		m.authMethod = auth.MethodExisting
		if cfg.Auth.AuthURL != "" {
			m.authURL = cfg.Auth.AuthURL
		}
		m.tier = cfg.Local.Tier
		if cfg.Local.VaultPath != "" && m.setupVaultInput != nil {
			m.setupVaultInput.SetValue(cfg.Local.VaultPath)
		}
		m.state = stateRerun
		m.rerunCursor = rerunReAudit // safe default — non-destructive action
		return m, nil
	}

	// User passed --tier — bypass picker.
	if m.tier.Valid() {
		return m.startAuth()
	}

	// Fresh install (or tier never recorded) — show the picker.
	m.state = stateTier
	m.tierCursor = tierChoiceFull // default highlight on Tier 1
	return m, nil
}

// confirmTier commits the tier choice from the picker screen and transitions to Auth.
func (m Model) confirmTier() (tea.Model, tea.Cmd) {
	if m.tierCursor == tierChoiceRemote {
		m.tier = config.TierRemote
	} else {
		m.tier = config.TierFull
	}
	return m.startAuth()
}

// confirmRerun executes the selected re-run action.
func (m Model) confirmRerun() (tea.Model, tea.Cmd) {
	switch m.rerunCursor {
	case rerunChangeTier:
		// Go to the tier picker; selecting a tier there will re-run auth-skip + audit.
		m.state = stateTier
		// Default-highlight the *opposite* of current tier so the user can Enter through quickly.
		if m.tier == config.TierFull {
			m.tierCursor = tierChoiceRemote
		} else {
			m.tierCursor = tierChoiceFull
		}
		return m, nil

	case rerunReAudit:
		// Jump to audit; auth/tier are already known from config.
		m.state = stateAudit
		return m, m.runAuditCmd()

	case rerunRewriteMCP:
		// Re-write .mcp.json in-place based on current tier/config, then show confirmation.
		if err := m.rewriteMCPFromConfig(); err != nil {
			m.rerunMessage = fmt.Sprintf("MCP rewrite failed: %v", err)
			m.rerunMessageErr = true
		} else {
			m.rerunMessage = "MCP config rewritten successfully."
			m.rerunMessageErr = false
		}
		// Stay on the re-run screen so the user sees the confirmation line.
		return m, nil

	case rerunExit:
		return m, tea.Quit
	}

	return m, nil
}

// --- Auth flow helpers ---

// startAuth transitions to the auth screen, or skips it if already configured.
func (m Model) startAuth() (tea.Model, tea.Cmd) {
	// Check for existing valid API key — skip auth if found.
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Auth.APIKey != "" && strings.HasPrefix(cfg.Auth.APIKey, "dck_") {
		m.apiKey = cfg.Auth.APIKey
		m.authMethod = auth.MethodExisting
		m.authURL = cfg.Auth.AuthURL
		if m.authURL == "" {
			m.authURL = "https://auth.jules.solutions"
		}
		// Jump straight to audit.
		m.state = stateAudit
		return m, m.runAuditCmd()
	}

	// No existing key — run normal auth flow.
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

// runInstallCmd runs auto-install for all missing tools in the background.
// Tier-aware: Tier 2 only auto-installs tools it actually needs.
func (m Model) runInstallCmd() tea.Cmd {
	checks := m.auditResults
	tier := string(m.tier)
	return func() tea.Msg {
		results := audit.InstallMissingForTier(checks, tier)
		return installDoneMsg{results: results}
	}
}

// finishSetup starts the vault download after setup questions are answered.
func (m Model) finishSetup() (tea.Model, tea.Cmd) {
	// Save config immediately (vault path + API key).
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

	// Start vault download.
	m.state = stateDownload
	return m, m.runVaultDownloadCmd(vaultPath)
}

// runVaultDownloadCmd attempts to clone or scaffold the vault (Tier 1 only —
// Tier 2 never calls this path).
func (m Model) runVaultDownloadCmd(vaultPath string) tea.Cmd {
	params := setup.ScaffoldParams{
		APIKey: m.apiKey,
		// Use the production API URL, not the auth URL (they're different services).
		APIURL: "https://api.jules.solutions",
		// /sse suffix required — Claude Code's SSE transport reads the full path.
		MCPURL: defaultMCPURL(m),
	}
	// Derive username/vault name from the vault path.
	params.VaultName = filepath.Base(vaultPath)
	if len(params.VaultName) > 5 && params.VaultName[len(params.VaultName)-5:] == ".Life" {
		params.UserName = params.VaultName[:len(params.VaultName)-5]
	} else {
		params.UserName = params.VaultName
	}

	return func() tea.Msg {
		method, err := setup.DownloadVaultWithParams(vaultPath, params)
		return vaultDownloadMsg{method: method, err: err}
	}
}

// writeConfigAndFinish is the Tier 1 tail: write vault-root .mcp.json, install
// jules-local CLI, and advance to Done. Tier 2 goes through finishTier2 instead.
func (m Model) writeConfigAndFinish() (tea.Model, tea.Cmd) {
	vaultPath := ""
	if m.setupVaultInput != nil {
		vaultPath = m.setupVaultInput.Value()
	}

	// Write .mcp.json (direct SSE, per Q5 decision: unified shape across tiers).
	// Best-effort — don't block on failure.
	if m.setupConfigMCP && vaultPath != "" {
		mcpPath, err := setup.WriteMCPConfigForTier(config.TierFull, vaultPath, m.apiKey, defaultMCPURL(m))
		if err == nil {
			m.mcpPathWritten = mcpPath
			_ = persistMCPPath(mcpPath)
		}
	}

	// Install jules-local (the platform CLI: `jules`, `js`, vault ops, playgrounds).
	// Tier 1's whole value proposition is the full local runtime. No tier gating here.
	if err := setup.InstallJulesLocal(); err != nil {
		// Non-fatal: user can install manually later. Log the error.
		m.installLocalErr = err
	}

	m.state = stateDone
	return m, nil
}

// finishTier2 is the Tier 2 tail: write the user-global .mcp.json (embedded
// X-API-Key pointing at the remote SSE endpoint) and advance to Done. No vault,
// no jules-local install.
func (m Model) finishTier2() (tea.Model, tea.Cmd) {
	mcpPath, err := setup.WriteMCPConfigForTier(config.TierRemote, "", m.apiKey, defaultMCPURL(m))
	if err != nil {
		m.err = fmt.Errorf("writing MCP config: %w", err)
		m.state = stateError
		return m, nil
	}
	m.mcpPathWritten = mcpPath
	_ = persistMCPPath(mcpPath)

	// Mark the download method as a no-op so the Done screen renders sensibly.
	m.vaultDownloadMethod = "skipped_tier2"
	m.state = stateDone
	return m, nil
}

// defaultMCPURL resolves the MCP SSE URL from config (falling back to the
// canonical production endpoint).
func defaultMCPURL(m Model) string {
	cfg, _ := config.LoadConfig()
	if cfg.Auth.MCPURL != "" {
		return cfg.Auth.MCPURL
	}
	return "https://mcp.jules.solutions/sse"
}

// persistMCPPath records the written MCP path in config.toml so re-runs can
// find it (or the Done screen can show it).
func persistMCPPath(mcpPath string) error {
	cfg, _ := config.LoadConfig()
	cfg.Local.MCPPath = mcpPath
	return config.SaveConfig(cfg)
}

// rewriteMCPFromConfig is called from the re-run menu. It regenerates the
// .mcp.json file at the tier-appropriate location using current config values.
func (m Model) rewriteMCPFromConfig() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if !cfg.Local.Tier.Valid() {
		return fmt.Errorf("tier not set in config — re-run installer without --resume to choose one")
	}
	if cfg.Auth.APIKey == "" {
		return fmt.Errorf("no API key in config — run installer to authenticate")
	}
	mcpURL := cfg.Auth.MCPURL
	if mcpURL == "" {
		mcpURL = "https://mcp.jules.solutions/sse"
	}
	_, err = setup.WriteMCPConfigForTier(cfg.Local.Tier, cfg.Local.VaultPath, cfg.Auth.APIKey, mcpURL)
	return err
}

// applyResume applies the detected resume state, loading config and jumping to the
// appropriate step.
func (m Model) applyResume(rs resumeState) (tea.Model, tea.Cmd) {
	if rs == resumeFromWelcome {
		// Nothing to skip — stay on welcome screen.
		return m, nil
	}

	// Load config to populate model fields.
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Auth.APIKey != "" {
		m.apiKey = cfg.Auth.APIKey
		m.authMethod = auth.MethodExisting
		if cfg.Auth.AuthURL != "" {
			m.authURL = cfg.Auth.AuthURL
		}
		if cfg.Local.VaultPath != "" && m.setupVaultInput != nil {
			m.setupVaultInput.SetValue(cfg.Local.VaultPath)
		}
		// Recover tier from config so resume behaves tier-appropriately.
		// If a --tier flag was passed, the caller's value wins over the
		// config (lets the user switch tier via resume + flag).
		if m.tier == "" && cfg.Local.Tier.Valid() {
			m.tier = cfg.Local.Tier
		}
	}

	switch rs {
	case resumeFromAudit:
		m.state = stateAudit
		return m, m.runAuditCmd()

	case resumeFromSetup:
		m.state = stateAudit
		m.auditSubState = auditShowResults
		return m, m.runAuditCmd()

	case resumeFromDownload:
		vaultPath := ""
		if m.setupVaultInput != nil {
			vaultPath = m.setupVaultInput.Value()
		}
		m.state = stateDownload
		return m, m.runVaultDownloadCmd(vaultPath)

	case resumeFromDone:
		// Mark vault as "existing" since it's already there.
		m.vaultDownloadMethod = "existing"
		m.state = stateDone
		return m, nil
	}

	return m, nil
}

// runLaunchCmd attempts to launch Claude Code in the vault directory.
func (m Model) runLaunchCmd() tea.Cmd {
	vaultPath := ""
	if m.setupVaultInput != nil {
		vaultPath = m.setupVaultInput.Value()
	}
	return func() tea.Msg {
		err := setup.LaunchClaudeCode(vaultPath)
		return launchDoneMsg{err: err}
	}
}

// tickCmd returns a command that fires a tickMsg after a short interval,
// driving the spinner animation.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
