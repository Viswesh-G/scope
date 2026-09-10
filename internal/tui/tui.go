// Package tui provides an interactive terminal user interface for SCP.
// We use the Bubbletea framework, which follows the Elm architecture (Model-View-Update).
//
// Instead of suspending the TUI and dumping text to the raw terminal, we run
// the search command in the background with the --json flag. We capture the
// JSON output, parse it, and display it beautifully in a "Results" tab.
package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Viswesh-G/scope/internal/search"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The three main tabs in our application
const (
	tabSearch = iota
	tabResults
	tabHelp
)

// model holds all the state for our terminal application.
type model struct {
	activeTab int

	// Search input state
	queryInput string
	pathInput  string
	flagsInput string
	inputFocus int // 0 = query, 1 = path, 2 = flags

	// Results state
	isSearching bool
	matches     []search.JSONMatch
	scrollPos   int

	// Status message for the bottom bar
	statusMsg string

	// Terminal dimensions (updated dynamically on resize)
	width  int
	height int
}

// searchResultMsg is the message we send back to the Update function
// when a background search finishes.
type searchResultMsg struct {
	matches []search.JSONMatch
	err     error
}

// InitialModel sets up the default starting state.
// It can optionally accept pre-loaded matches (e.g. piped from standard input).
func InitialModel(matches []search.JSONMatch) model {
	activeTab := tabSearch
	statusMsg := "Ready. Enter a pattern to begin."

	// If we received piped matches, jump straight to the results!
	if len(matches) > 0 {
		activeTab = tabResults
		statusMsg = fmt.Sprintf("Loaded %d piped matches.", len(matches))
	}

	return model{
		activeTab:  activeTab,
		queryInput: "",
		pathInput:  ".",
		flagsInput: "",
		inputFocus: 0,
		statusMsg:  statusMsg,
		matches:    matches,
	}
}

// Init can return a command to run right when the application starts.
// We don't need to do anything special on startup.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles all incoming events (like keyboard presses or background tasks finishing).
// It returns the updated model and any new commands to run.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// The user resized the terminal window
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// Background search finished!
	case searchResultMsg:
		m.isSearching = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.matches = msg.matches
			m.activeTab = tabResults // automatically switch to results tab
			m.scrollPos = 0
			m.statusMsg = fmt.Sprintf("Found %d matches.", len(m.matches))
		}
		return m, nil

	// A keyboard key was pressed
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			m.activeTab = (m.activeTab + 1) % 3 // cycle through the 3 tabs
			return m, nil

		case "up":
			if m.activeTab == tabSearch {
				m.inputFocus--
				if m.inputFocus < 0 {
					m.inputFocus = 2
				}
			} else if m.activeTab == tabResults && m.scrollPos > 0 {
				m.scrollPos--
			}
			return m, nil

		case "down":
			if m.activeTab == tabSearch {
				m.inputFocus = (m.inputFocus + 1) % 3
			} else if m.activeTab == tabResults && m.scrollPos < len(m.matches)-1 {
				m.scrollPos++
			}
			return m, nil

		case "enter":
			if m.activeTab == tabSearch {
				if m.queryInput != "" {
					m.isSearching = true
					m.statusMsg = "Searching..."
					return m, runSearchCmd(m.queryInput, m.pathInput, m.flagsInput)
				}
				m.statusMsg = "Error: Pattern cannot be empty"
			}

		case "backspace":
			if m.activeTab == tabSearch {
				if m.inputFocus == 0 && len(m.queryInput) > 0 {
					m.queryInput = m.queryInput[:len(m.queryInput)-1]
				} else if m.inputFocus == 1 && len(m.pathInput) > 0 {
					m.pathInput = m.pathInput[:len(m.pathInput)-1]
				} else if m.inputFocus == 2 && len(m.flagsInput) > 0 {
					m.flagsInput = m.flagsInput[:len(m.flagsInput)-1]
				}
			}

		default:
			// Normal typing in the search tab
			if m.activeTab == tabSearch && len(msg.String()) == 1 {
				if m.inputFocus == 0 {
					m.queryInput += msg.String()
				} else if m.inputFocus == 1 {
					m.pathInput += msg.String()
				} else if m.inputFocus == 2 {
					m.flagsInput += msg.String()
				}
			}
		}
	}

	return m, nil
}

// runSearchCmd spawns a background goroutine that runs the scp binary in JSON mode.
// We capture the output without suspending the TUI.
func runSearchCmd(pattern, path, flags string) tea.Cmd {
	return func() tea.Msg {
		exe, err := os.Executable()
		if err != nil {
			return searchResultMsg{err: fmt.Errorf("finding binary: %v", err)}
		}

		args := []string{"search", "-p", pattern, "--path", path, "--json", "-q"}
		
		if strings.TrimSpace(flags) != "" {
			var current string
			var inQuote bool
			var parsed []string
			for _, r := range flags {
				if r == '"' || r == '\'' {
					inQuote = !inQuote
					continue
				}
				if r == ' ' && !inQuote {
					if current != "" {
						parsed = append(parsed, current)
						current = ""
					}
				} else {
					current += string(r)
				}
			}
			if current != "" {
				parsed = append(parsed, current)
			}
			args = append(args, parsed...)
		}

		c := exec.Command(exe, args...)
		
		// Run the command and capture stdout
		out, err := c.Output()
		if err != nil {
			// exec.ExitError means the command ran but returned non-zero (which ripgrep clones often do if no matches are found)
			// But if it's not an ExitError, the command failed to start at all.
			if _, isExitError := err.(*exec.ExitError); !isExitError {
				return searchResultMsg{err: fmt.Errorf("execution failed: %v", err)}
			}
		}

		var matches []search.JSONMatch
		if len(out) > 0 {
			if err := json.Unmarshal(out, &matches); err != nil {
				return searchResultMsg{err: fmt.Errorf("parsing JSON: %v", err)}
			}
		}

		return searchResultMsg{matches: matches}
	}
}

// -- View / Lipgloss styling --

var (
	colorAccent = lipgloss.Color("42") // Green
	colorDim    = lipgloss.Color("240")
	colorWhite  = lipgloss.Color("255")
	colorRed    = lipgloss.Color("196")
	colorBlue   = lipgloss.Color("39")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	styleTabActive = lipgloss.NewStyle().
			Foreground(colorAccent).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorAccent).
			Padding(0, 2)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorDim).
				Padding(0, 2)

	styleInputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	// Note: We avoid styleInputBox.Copy() because it was deprecated in lipgloss.
	// Instead we declare a fresh style with the same base properties.
	styleInputBoxActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)
)

// View renders the entire screen based on the current model state.
func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// 1. Header & Tabs
	title := styleTitle.Render("SCP Interactive Terminal")

	tabStrs := []string{"Search", "Results", "Help"}
	var renderedTabs []string
	for i, t := range tabStrs {
		if i == m.activeTab {
			renderedTabs = append(renderedTabs, styleTabActive.Render(t))
		} else {
			renderedTabs = append(renderedTabs, styleTabInactive.Render(t))
		}
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// 2. Main Content Area
	var content string

	switch m.activeTab {
	case tabSearch:
		// Draw the three input boxes (Pattern, Path, Flags)
		qStyle := styleInputBox
		pStyle := styleInputBox
		fStyle := styleInputBox

		qCursor := ""
		pCursor := ""
		fCursor := ""
		if m.inputFocus == 0 {
			qStyle = styleInputBoxActive
			qCursor = "█"
		} else if m.inputFocus == 1 {
			pStyle = styleInputBoxActive
			pCursor = "█"
		} else {
			fStyle = styleInputBoxActive
			fCursor = "█"
		}

		qBox := qStyle.Render(fmt.Sprintf("Pattern: %s%s", m.queryInput, qCursor))
		pBox := pStyle.Render(fmt.Sprintf("Path:    %s%s", m.pathInput, pCursor))
		fBox := fStyle.Render(fmt.Sprintf("Flags:   %s%s", m.flagsInput, fCursor))

		content = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().MarginBottom(1).Render("Enter search criteria:"),
			qBox,
			pBox,
			fBox,
			lipgloss.NewStyle().MarginTop(1).Foreground(colorDim).Render("Use UP/DOWN to switch fields. Press ENTER to search.\nFlags: Any CLI flags (e.g. -i -C 2 -g *.go)"),
		)

	case tabResults:
		// Draw the list of JSON matches
		if m.isSearching {
			content = lipgloss.NewStyle().Foreground(colorAccent).Render("Searching... Please wait.")
		} else if len(m.matches) == 0 {
			content = lipgloss.NewStyle().Foreground(colorDim).Render("No results to display.")
		} else {
			// We only show the matches that fit on the screen based on the scroll position
			// A real app would use the bubbletea 'viewport' component for this, but this works nicely for a demo!
			var lines []string
			
			// Figure out how many lines we can safely print
			maxVisible := m.height - 10 
			if maxVisible < 1 {
				maxVisible = 1
			}

			start := m.scrollPos
			end := start + maxVisible
			if end > len(m.matches) {
				end = len(m.matches)
			}

			for i := start; i < end; i++ {
				match := m.matches[i]
				
				// Truncate really long lines so they don't break the layout
				lineContent := match.Content
				if len(lineContent) > 80 {
					lineContent = lineContent[:77] + "..."
				}

				formatted := fmt.Sprintf("%s:%s %s",
					lipgloss.NewStyle().Foreground(colorBlue).Render(match.File),
					lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d", match.LineNum)),
					lineContent,
				)
				lines = append(lines, formatted)
			}

			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Foreground(colorDim).Render(
				fmt.Sprintf("Showing %d-%d of %d (Use UP/DOWN to scroll)", start+1, end, len(m.matches)),
			))

			content = lipgloss.JoinVertical(lipgloss.Left, lines...)
		}

	case tabHelp:
		content = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("SCP Keybindings"),
			"",
			"TAB       : Switch tabs",
			"UP/DOWN   : Switch input fields / Scroll results",
			"ENTER     : Execute search",
			"ESC/CTRL+C: Quit",
		)
	}

	// 3. Status Bar
	statusColor := colorDim
	if strings.HasPrefix(m.statusMsg, "Error") {
		statusColor = colorRed
	} else if strings.Contains(m.statusMsg, "Found") || strings.Contains(m.statusMsg, "Searching") {
		statusColor = colorAccent
	}
	statusBar := lipgloss.NewStyle().Foreground(statusColor).Render(m.statusMsg)

	// Assemble everything vertically
	view := lipgloss.JoinVertical(lipgloss.Left,
		title,
		tabs,
		"",
		content,
		"",
		"────────────────────────────",
		statusBar,
	)

	return lipgloss.NewStyle().Margin(1, 2).Render(view)
}

// StartTUI initializes and runs the tea program.
func StartTUI() error {
	var initialMatches []search.JSONMatch
	var input *os.File = os.Stdin

	// Check if data is piped into stdin
	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		// Read from pipe
		bytes, err := io.ReadAll(os.Stdin)
		if err == nil && len(bytes) > 0 {
			_ = json.Unmarshal(bytes, &initialMatches)
		}
		
		// Redirect Bubbletea input to the terminal keyboard so the user can still interact
		var tty string
		if runtime.GOOS == "windows" {
			tty = "CONIN$"
		} else {
			tty = "/dev/tty"
		}
		f, err := os.Open(tty)
		if err == nil {
			input = f
		}
	}

	p := tea.NewProgram(InitialModel(initialMatches), tea.WithAltScreen(), tea.WithInput(input))
	_, err := p.Run()
	return err
}
