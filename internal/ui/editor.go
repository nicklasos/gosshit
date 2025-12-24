package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicklasos/gosshit/internal/sshconfig"
)

// EditorModel represents the form-based editor for host entries
type EditorModel struct {
	fields       []textinput.Model
	focused      int
	entry        *sshconfig.HostEntry
	isNew        bool
	width        int
	height       int
	errorMsg     string
	keySelector  *KeySelectorModel
	selectingKey bool
	viewport     viewport.Model
}

// Field indices
const (
	fieldHost = iota
	fieldHostName
	fieldUser
	fieldPort
	fieldIdentityFile
	fieldDescription
	fieldTags
	fieldCount
)

// NewEditorModel creates a new editor model
func NewEditorModel() *EditorModel {
	m := &EditorModel{
		fields:      make([]textinput.Model, fieldCount),
		keySelector: NewKeySelectorModel(),
		viewport:    viewport.New(0, 0),
	}

	// Initialize fields
	m.fields[fieldHost] = textinput.New()
	m.fields[fieldHost].Placeholder = "host-alias"
	m.fields[fieldHost].Focus()

	m.fields[fieldHostName] = textinput.New()
	m.fields[fieldHostName].Placeholder = "example.com or 192.168.1.1"

	m.fields[fieldUser] = textinput.New()
	m.fields[fieldUser].Placeholder = "root"

	m.fields[fieldPort] = textinput.New()
	m.fields[fieldPort].Placeholder = "22"

	m.fields[fieldIdentityFile] = textinput.New()
	m.fields[fieldIdentityFile].Placeholder = "~/.ssh/id_rsa (optional - enter path manually)"

	m.fields[fieldDescription] = textinput.New()
	m.fields[fieldDescription].Placeholder = "Description (optional)"

	m.fields[fieldTags] = textinput.New()
	m.fields[fieldTags].Placeholder = "prod,dev,stage (comma-separated, optional)"

	return m
}

// Init initializes the editor model
func (m *EditorModel) Init() tea.Cmd {
	return textinput.Blink
}

// SetEntry sets the entry to edit (nil for new entry)
func (m *EditorModel) SetEntry(entry *sshconfig.HostEntry) {
	m.entry = entry
	m.isNew = entry == nil
	m.errorMsg = ""

	if entry != nil {
		m.fields[fieldHost].SetValue(entry.Host)
		m.fields[fieldHostName].SetValue(entry.HostName)
		m.fields[fieldUser].SetValue(entry.User)
		m.fields[fieldPort].SetValue(entry.Port)
		m.fields[fieldIdentityFile].SetValue(entry.IdentityFile)
		m.fields[fieldDescription].SetValue(entry.Description)
		// Convert tags slice to comma-separated string
		if len(entry.Tags) > 0 {
			m.fields[fieldTags].SetValue(strings.Join(entry.Tags, ", "))
		} else {
			m.fields[fieldTags].SetValue("")
		}
	} else {
		// Default values for new entries
		m.fields[fieldHost].SetValue("")
		m.fields[fieldHostName].SetValue("")
		m.fields[fieldUser].SetValue("root")
		m.fields[fieldPort].SetValue("22")
		m.fields[fieldIdentityFile].SetValue("")
		m.fields[fieldDescription].SetValue("")
		m.fields[fieldTags].SetValue("")
	}

	// Focus first field
	m.focused = 0
	m.updateFocus()
}

// SetSize sets the size of the editor
func (m *EditorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Update field widths to match editor width
	fieldWidth := width - 20 // Leave space for padding and borders
	for i := range m.fields {
		m.fields[i].Width = fieldWidth
	}
	m.keySelector.SetSize(width, height)
	// Set viewport size (accounting for borders - 2 lines top/bottom)
	m.viewport.Width = width - 4
	m.viewport.Height = height - 4
	// Initialize viewport content
	m.updateViewportContent()
}

// Update handles updates to the editor model
func (m *EditorModel) Update(msg tea.Msg) (*EditorModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case keyLoadResult, keyLoadError:
		// Handle key selector messages
		if m.selectingKey {
			selector, cmd := m.keySelector.Update(msg)
			m.keySelector = selector
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if !m.keySelector.IsOpen() {
				m.selectingKey = false
			}
			return m, tea.Batch(cmds...)
		}

	case keySelectedMsg:
		// Key was selected, set it in the IdentityFile field
		if msg.key != "" {
			m.fields[fieldIdentityFile].SetValue(msg.key)
		}
		m.selectingKey = false
		return m, nil

	case tea.KeyMsg:
		// If key selector is open, handle it first
		if m.selectingKey {
			selector, cmd := m.keySelector.Update(msg)
			m.keySelector = selector
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if !m.keySelector.IsOpen() {
				m.selectingKey = false
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "tab":
			m.focused = (m.focused + 1) % fieldCount
			m.updateFocus()
			return m, nil
		case "shift+tab":
			m.focused = (m.focused - 1 + fieldCount) % fieldCount
			m.updateFocus()
			return m, nil
		case "enter":
			// Will be handled by parent model
			return m, nil
		case "esc":
			// Will be handled by parent model
			return m, nil
		}
	}

	// Update focused field first (before viewport, so content is up to date)
	var fieldCmd tea.Cmd
	m.fields[m.focused], fieldCmd = m.fields[m.focused].Update(msg)
	if fieldCmd != nil {
		cmds = append(cmds, fieldCmd)
	}

	// Handle viewport scrolling (only if viewport is initialized)
	if m.viewport.Height > 0 {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		if vpCmd != nil {
			cmds = append(cmds, vpCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateFocus updates which field is focused
func (m *EditorModel) updateFocus() {
	for i := range m.fields {
		if i == m.focused {
			m.fields[i].Focus()
		} else {
			m.fields[i].Blur()
		}
	}
}

// Validate validates the form fields
func (m *EditorModel) Validate() error {
	host := m.fields[fieldHost].Value()

	if host == "" {
		return fmt.Errorf("Host alias is required")
	}
	// Host * entries don't need HostName
	if host != "*" {
		hostname := m.fields[fieldHostName].Value()
		if hostname == "" {
			return fmt.Errorf("HostName is required")
		}
	}

	return nil
}

// GetEntry returns the entry from the form fields
func (m *EditorModel) GetEntry() *sshconfig.HostEntry {
	// Parse tags from comma-separated string
	var tags []string
	tagsStr := strings.TrimSpace(m.fields[fieldTags].Value())
	if tagsStr != "" {
		parts := strings.Split(tagsStr, ",")
		for _, tag := range parts {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	return &sshconfig.HostEntry{
		Host:         m.fields[fieldHost].Value(),
		HostName:     m.fields[fieldHostName].Value(),
		User:         m.fields[fieldUser].Value(),
		Port:         m.fields[fieldPort].Value(),
		IdentityFile: m.fields[fieldIdentityFile].Value(),
		Description:  m.fields[fieldDescription].Value(),
		Tags:         tags,
	}
}

// SetError sets an error message
func (m *EditorModel) SetError(msg string) {
	m.errorMsg = msg
}

// updateViewportContent updates the viewport with the current form content
func (m *EditorModel) updateViewportContent() {
	var lines []string

	// Title/Header
	title := "Edit Host"
	if m.isNew {
		title = "Add New Host"
	}
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")

	// Field labels
	labels := []string{"Host:", "HostName:", "User:", "Port:", "IdentityFile:", "Description:", "Tags:"}
	for i, label := range labels {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render(label))

		var fieldView string
		if i == m.focused {
			fieldView = inputFocusedStyle.Render(m.fields[i].View())
		} else {
			fieldView = inputStyle.Render(m.fields[i].View())
		}
		lines = append(lines, fieldView)
	}

	// Error message
	if m.errorMsg != "" {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("Error: "+m.errorMsg))
	}

	// Help text
	lines = append(lines, "")
	helpText := "Tab: next field | Shift+Tab: previous field | Enter: save | Esc: cancel | ↑↓: scroll"
	lines = append(lines, helpStyle.Render(helpText))

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)

	// Scroll to show focused field (each field takes ~3 lines: empty line + label + input)
	// Account for title and empty line at top (2 lines)
	if m.focused >= 0 && m.focused < len(labels) {
		// Calculate approximate line number of focused field (1-indexed for content)
		// Title takes 1 line, empty line takes 1 line, then fields start
		estimatedLine := 2 + m.focused*3 + 1

		// Check if focused field is above visible area
		if estimatedLine < m.viewport.YOffset {
			// Scroll to show focused field at top
			m.viewport.SetYOffset(estimatedLine - 1)
		} else if estimatedLine+2 > m.viewport.YOffset+m.viewport.Height {
			// Scroll to show focused field at bottom (field takes ~3 lines)
			m.viewport.SetYOffset(estimatedLine + 2 - m.viewport.Height)
		}
	}
}

// View renders the editor view
func (m *EditorModel) View() string {
	// Update viewport content
	m.updateViewportContent()

	// Render viewport inside panel
	view := detailPanelStyle.Width(m.width).Height(m.height).Render(m.viewport.View())

	// Show key selector on top if open
	if m.selectingKey {
		selectorView := m.keySelector.View()
		// Overlay the selector
		return lipgloss.JoinVertical(lipgloss.Center, selectorView, view)
	}

	// Ensure the view fills the available space and shows borders properly
	return view
}
