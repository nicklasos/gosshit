package ui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// KeySelectorModel represents a file selector for SSH keys
type KeySelectorModel struct {
	keys     []string
	selected int
	width    int
	height   int
	isOpen   bool
}

// NewKeySelectorModel creates a new key selector model
func NewKeySelectorModel() *KeySelectorModel {
	return &KeySelectorModel{
		keys:     []string{},
		selected: 0,
		isOpen:   false,
	}
}

// Open opens the key selector and loads keys from ~/.ssh/
func (m *KeySelectorModel) Open() tea.Cmd {
	m.isOpen = true
	m.selected = 0
	return m.loadKeys()
}

// Close closes the key selector
func (m *KeySelectorModel) Close() {
	m.isOpen = false
	m.keys = []string{}
}

// IsOpen returns whether the selector is open
func (m *KeySelectorModel) IsOpen() bool {
	return m.isOpen
}

// loadKeys loads SSH key files from ~/.ssh/
func (m *KeySelectorModel) loadKeys() tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return keyLoadError{err: err}
		}

		sshDir := filepath.Join(homeDir, ".ssh")
		files, err := os.ReadDir(sshDir)
		if err != nil {
			return keyLoadError{err: err}
		}

		var keys []string
		// Common SSH key file patterns (exclude .pub files as we want private keys)
		for _, file := range files {
			name := file.Name()
			// Skip directories, .pub files, known_hosts, config, and other non-key files
			if file.IsDir() {
				continue
			}
			if strings.HasSuffix(name, ".pub") {
				continue
			}
			if name == "known_hosts" || name == "config" || name == "authorized_keys" {
				continue
			}
			// Include common key file patterns
			if strings.HasPrefix(name, "id_") || strings.HasPrefix(name, "key_") {
				// Convert to ~/.ssh/name format
				keys = append(keys, "~/.ssh/"+name)
			}
		}

		// Add option for custom path
		keys = append(keys, "(custom path)")

		return keyLoadResult{keys: keys}
	}
}

// keyLoadResult is a message sent when keys are loaded
type keyLoadResult struct {
	keys []string
}

// keyLoadError is a message sent when key loading fails
type keyLoadError struct {
	err error
}

// keySelectedMsg is sent when a key is selected
type keySelectedMsg struct {
	key string
}

// Export these types for use in editor
// These are defined here but used in editor.go

// Update handles updates to the key selector
func (m *KeySelectorModel) Update(msg tea.Msg) (*KeySelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case keyLoadResult:
		m.keys = msg.keys
		return m, nil

	case keyLoadError:
		// On error, just return empty list
		m.keys = []string{}
		return m, nil

	case tea.KeyMsg:
		if !m.isOpen {
			return m, nil
		}

		switch msg.String() {
		case "esc":
			m.Close()
			return m, nil

		case "j", "down":
			if m.selected < len(m.keys)-1 {
				m.selected++
			}
			return m, nil

		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case "enter":
			if m.selected >= 0 && m.selected < len(m.keys) {
				key := m.keys[m.selected]
				if key == "(custom path)" {
					key = ""
				}
				m.Close()
				return m, func() tea.Msg {
					return keySelectedMsg{key: key}
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// SetSize sets the size of the selector
func (m *KeySelectorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the key selector
func (m *KeySelectorModel) View() string {
	if !m.isOpen {
		return ""
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Select SSH Key"))

	if len(m.keys) == 0 {
		lines = append(lines, "")
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("No keys found in ~/.ssh/"))
		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("Esc: cancel"))
	} else {
		// Show keys list
		visibleCount := min(m.height-6, len(m.keys)) // Account for title, padding, help text
		start := max(0, m.selected-visibleCount/2)
		end := min(len(m.keys), start+visibleCount)

		for i := start; i < end; i++ {
			key := m.keys[i]
			if i == m.selected {
				lines = append(lines, listItemSelectedStyle.Render("▶ "+key))
			} else {
				lines = append(lines, listItemStyle.Render("  "+key))
			}
		}

		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("j/k: navigate | Enter: select | Esc: cancel"))
	}

	content := strings.Join(lines, "\n")
	// Use a simple overlay style for the selector
	selectorStyle := detailPanelStyle.Copy().
		Width(m.width).
		Height(m.height).
		BorderForeground(accentColor).
		Background(bgColor)
	return selectorStyle.Render(content)
}
