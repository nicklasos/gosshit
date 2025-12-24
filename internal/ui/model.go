package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicklasos/gosshit/internal/sshconfig"
	"github.com/nicklasos/gosshit/internal/storage"
)

// Mode represents the current UI mode
type Mode int

const (
	ModeList Mode = iota
	ModeSearch
	ModeEdit
	ModeAdd
	ModeDelete
	ModeClearVisits
)

// Model represents the main application model
type Model struct {
	listModel   *ListModel
	detailModel *DetailModel
	editorModel *EditorModel
	tracker     *storage.VisitTracker
	entries     []*sshconfig.HostEntry // Display entries (Host * filtered out)
	configPath  string

	mode          Mode
	searchInput   textinput.Model
	deleteConfirm bool

	width  int
	height int
	err    error
}

// InitialModel creates the initial model
func InitialModel(configPath string) (*Model, error) {
	// Load SSH config
	entries, _, err := sshconfig.ParseConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Filter out Host * entries from display (they're global config, not specific hosts)
	// But keep them in the entries list for preservation
	displayEntries := make([]*sshconfig.HostEntry, 0, len(entries))
	allEntries := make([]*sshconfig.HostEntry, 0, len(entries))
	for _, entry := range entries {
		allEntries = append(allEntries, entry)
		if entry.Host != "*" {
			displayEntries = append(displayEntries, entry)
		}
	}

	// Load visit tracker
	tracker, err := storage.NewVisitTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to load visit tracker: %w", err)
	}

	// Get visit counts (only for display entries)
	visitCounts := make(map[string]int)
	for _, entry := range displayEntries {
		visitCounts[entry.Host] = tracker.GetCount(entry.Host)
	}

	// Sort entries by visit count (only display entries)
	sortedHosts := tracker.SortByVisits(getHostNames(displayEntries))
	sortedEntries := sortEntriesByHosts(displayEntries, sortedHosts)

	// Initialize models
	listModel := NewListModel(sortedEntries, visitCounts)
	detailModel := NewDetailModel()
	editorModel := NewEditorModel()

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."

	model := &Model{
		listModel:     listModel,
		detailModel:   detailModel,
		editorModel:   editorModel,
		tracker:       tracker,
		entries:       sortedEntries, // Display entries (without Host *)
		configPath:    configPath,
		mode:          ModeList,
		searchInput:   searchInput,
		deleteConfirm: false,
	}

	// Set initial selected entry
	if len(sortedEntries) > 0 {
		model.updateDetailView()
	}

	return model, nil
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.listModel.Init(),
		m.editorModel.Init(),
		textinput.Blink,
	)
}

// Update handles updates
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case tea.KeyMsg:
		// Check for mode-specific key handling first
		handled, model, cmd := m.handleKeyPress(msg)
		if handled {
			return model, cmd
		}
		// If not handled by handleKeyPress, continue to mode-specific updates
		// msg is still available for mode handlers below
	}

	// Handle mode-specific updates
	switch m.mode {
	case ModeSearch:
		// In search mode, let the search input handle updates
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.listModel.SetSearchTerm(m.searchInput.Value())
		m.updateDetailView()
		return m, cmd

	case ModeEdit, ModeAdd:
		var cmd tea.Cmd
		var updatedEditor *EditorModel
		updatedEditor, cmd = m.editorModel.Update(msg)
		m.editorModel = updatedEditor
		return m, cmd
	}

	// List mode updates
	var cmd tea.Cmd
	updatedList, listCmd := m.listModel.Update(msg)
	m.listModel = updatedList
	m.updateDetailView()
	return m, tea.Batch(cmd, listCmd)
}

// handleKeyPress handles key presses based on mode
// Returns (handled bool, model, cmd)
func (m *Model) handleKeyPress(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeSearch:
		// Only handle escape and enter to exit search mode
		// Other keys will be handled by the search input in Update
		if msg.String() == "esc" {
			m.mode = ModeList
			m.searchInput.SetValue("")
			m.listModel.SetSearchTerm("")
			m.searchInput.Blur()
			return true, m, nil
		}
		if msg.String() == "enter" {
			m.mode = ModeList
			m.searchInput.Blur()
			return true, m, nil
		}
		// Not handled here - let Update pass it to search input
		return false, m, nil

	case ModeEdit, ModeAdd:
		switch msg.String() {
		case "enter":
			if err := m.editorModel.Validate(); err != nil {
				m.editorModel.SetError(err.Error())
				return true, m, nil
			}
			model, cmd := m.saveEntry()
			return true, model, cmd
		case "esc":
			m.mode = ModeList
			m.editorModel.SetEntry(nil)
			return true, m, nil
		}
		return false, m, nil

	case ModeDelete:
		switch msg.String() {
		case "y", "Y":
			model, cmd := m.confirmDelete()
			return true, model, cmd
		case "n", "N", "esc":
			m.mode = ModeList
			m.deleteConfirm = false
			return true, m, nil
		}
		return false, m, nil

	case ModeClearVisits:
		switch msg.String() {
		case "y", "Y":
			model, cmd := m.confirmClearVisits()
			return true, model, cmd
		case "n", "N", "esc":
			m.mode = ModeList
			return true, m, nil
		}
		return false, m, nil

	case ModeList:
		handled, model, cmd := m.handleListKeyPress(msg)
		return handled, model, cmd
	}

	return false, m, nil
}

// handleListKeyPress handles key presses in list mode
func (m *Model) handleListKeyPress(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return true, m, tea.Quit

	case "j", "down":
		current := m.listModel.GetSelectedIndex()
		m.listModel.SetSelected(current + 1)
		m.updateDetailView()
		return true, m, nil

	case "k", "up":
		current := m.listModel.GetSelectedIndex()
		if current > 0 {
			m.listModel.SetSelected(current - 1)
		}
		m.updateDetailView()
		return true, m, nil

	case "/":
		m.mode = ModeSearch
		m.searchInput.Focus()
		return true, m, textinput.Blink

	case "a":
		m.mode = ModeAdd
		m.editorModel.SetEntry(nil)
		return true, m, nil

	case "e":
		entry := m.listModel.GetSelected()
		if entry != nil {
			m.mode = ModeEdit
			m.editorModel.SetEntry(entry)
		}
		return true, m, nil

	case "d":
		entry := m.listModel.GetSelected()
		if entry != nil {
			m.mode = ModeDelete
			m.deleteConfirm = false
		}
		return true, m, nil

	case "x":
		m.mode = ModeClearVisits
		return true, m, nil

	case "enter":
		entry := m.listModel.GetSelected()
		if entry != nil {
			model, cmd := m.connectToHost(entry)
			return true, model, cmd
		}
		return true, m, nil
	}

	return false, m, nil
}

// updateDetailView updates the detail view with the currently selected entry
func (m *Model) updateDetailView() {
	entry := m.listModel.GetSelected()
	if entry != nil {
		m.detailModel.SetEntry(entry)
		m.detailModel.SetVisitCount(m.tracker.GetCount(entry.Host))
	}
}

// updateSizes updates the sizes of all UI components
func (m *Model) updateSizes() {
	listWidth := 40
	detailWidth := m.width - listWidth - 6
	height := m.height - 4

	m.listModel.SetSize(listWidth, height)
	m.detailModel.SetSize(detailWidth, height)
	// Editor needs space for borders and padding, similar to other panels
	// Reduce by a bit to ensure borders are visible
	m.editorModel.SetSize(m.width-4, m.height-4)
}

// saveEntry saves the current entry from the editor
func (m *Model) saveEntry() (tea.Model, tea.Cmd) {
	entry := m.editorModel.GetEntry()
	var err error

	if m.mode == ModeAdd {
		err = sshconfig.AddEntry(m.configPath, entry)
	} else {
		oldEntry := m.editorModel.entry
		if oldEntry != nil {
			err = sshconfig.UpdateEntry(m.configPath, oldEntry.Host, entry)
		}
	}

	if err != nil {
		m.editorModel.SetError(err.Error())
		return m, nil
	}

	// Reload config
	allNewEntries, _, err := sshconfig.ParseConfig(m.configPath)
	if err != nil {
		m.err = err
		return m, nil
	}

	// Filter out Host * entries from display
	displayEntries := make([]*sshconfig.HostEntry, 0, len(allNewEntries))
	for _, e := range allNewEntries {
		if e.Host != "*" {
			displayEntries = append(displayEntries, e)
		}
	}

	// Get visit counts and sort (only for display entries)
	visitCounts := make(map[string]int)
	for _, e := range displayEntries {
		visitCounts[e.Host] = m.tracker.GetCount(e.Host)
	}
	sortedHosts := m.tracker.SortByVisits(getHostNames(displayEntries))
	sortedEntries := sortEntriesByHosts(displayEntries, sortedHosts)

	m.entries = sortedEntries
	m.listModel.SetEntries(sortedEntries)
	m.listModel.SetVisitCounts(visitCounts)
	m.mode = ModeList
	m.editorModel.SetEntry(nil)

	// Select the saved entry
	for i, e := range sortedEntries {
		if e.Host == entry.Host {
			m.listModel.SetSelected(i)
			break
		}
	}

	m.updateDetailView()
	return m, nil
}

// confirmDelete confirms and deletes the selected entry
func (m *Model) confirmDelete() (tea.Model, tea.Cmd) {
	entry := m.listModel.GetSelected()
	if entry == nil {
		m.mode = ModeList
		return m, nil
	}

	err := sshconfig.DeleteEntry(m.configPath, entry.Host)
	if err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	// Reload config
	allNewEntries, _, err := sshconfig.ParseConfig(m.configPath)
	if err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	// Filter out Host * entries from display
	displayEntries := make([]*sshconfig.HostEntry, 0, len(allNewEntries))
	for _, e := range allNewEntries {
		if e.Host != "*" {
			displayEntries = append(displayEntries, e)
		}
	}

	// Get visit counts and sort (only for display entries)
	visitCounts := make(map[string]int)
	for _, e := range displayEntries {
		visitCounts[e.Host] = m.tracker.GetCount(e.Host)
	}
	sortedHosts := m.tracker.SortByVisits(getHostNames(displayEntries))
	sortedEntries := sortEntriesByHosts(displayEntries, sortedHosts)

	m.entries = sortedEntries
	m.listModel.SetEntries(sortedEntries)
	m.listModel.SetVisitCounts(visitCounts)
	m.mode = ModeList
	m.deleteConfirm = false

	// Adjust selection
	current := m.listModel.GetSelectedIndex()
	if current >= len(sortedEntries) && len(sortedEntries) > 0 {
		m.listModel.SetSelected(len(sortedEntries) - 1)
	} else if len(sortedEntries) == 0 {
		m.listModel.SetSelected(0)
	}
	m.updateDetailView()
	return m, nil
}

func (m *Model) confirmClearVisits() (tea.Model, tea.Cmd) {
	// Clear all visit counts and save to file
	err := m.tracker.ClearAll()
	if err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	// Re-sort entries (now they'll be in alphabetical order since all counts are 0)
	sortedHosts := m.tracker.SortByVisits(getHostNames(m.entries))
	sortedEntries := sortEntriesByHosts(m.entries, sortedHosts)

	// Reset visit counts display
	visitCounts := make(map[string]int)
	for _, e := range sortedEntries {
		visitCounts[e.Host] = 0
	}

	m.entries = sortedEntries
	m.listModel.SetEntries(sortedEntries)
	m.listModel.SetVisitCounts(visitCounts)
	m.listModel.SetSelected(0)
	if len(sortedEntries) > 0 {
		m.updateDetailView()
	}

	m.mode = ModeList
	return m, nil
}

// connectToHost connects to the selected host via SSH
func (m *Model) connectToHost(entry *sshconfig.HostEntry) (tea.Model, tea.Cmd) {
	// Increment visit count
	m.tracker.Increment(entry.Host)
	if err := m.tracker.Save(); err != nil {
		m.err = err
		return m, nil
	}

	// Build SSH command
	cmd := exec.Command("ssh", entry.Host)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return tea.Quit()
	})
}

// View renders the model
func (m *Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.mode {
	case ModeSearch:
		return m.renderSearch()
	case ModeEdit, ModeAdd:
		return m.renderEditor()
	case ModeDelete:
		return m.renderDeleteConfirm()
	case ModeClearVisits:
		return m.renderClearVisitsConfirm()
	default:
		return m.renderList()
	}
}

// renderList renders the list view
func (m *Model) renderList() string {
	listView := m.listModel.View()
	detailView := m.detailModel.View()

	content := lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)

	// Status bar
	status := lipgloss.NewStyle().
		Foreground(fgColor).
		Padding(0, 1).
		Render("j/k: navigate | /: search | a: add | e: edit | d: delete | x: clear visits | enter: connect | q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

// renderSearch renders the search view
func (m *Model) renderSearch() string {
	listView := m.listModel.View()
	detailView := m.detailModel.View()

	content := lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)

	// Status bar with search query
	searchQuery := m.searchInput.Value()
	if searchQuery == "" {
		searchQuery = "(empty)"
	}
	status := lipgloss.NewStyle().
		Foreground(fgColor).
		Padding(0, 1).
		Render(fmt.Sprintf("Search: %s | Enter: select | Esc: cancel", searchQuery))

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

// renderEditor renders the editor view
func (m *Model) renderEditor() string {
	editorView := m.editorModel.View()
	// Add a newline at the top to push the editor down so top border is visible
	return "\n" + lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Top, editorView)
}

// renderDeleteConfirm renders the delete confirmation view
func (m *Model) renderDeleteConfirm() string {
	entry := m.listModel.GetSelected()
	if entry == nil {
		return ""
	}

	msg := fmt.Sprintf("Delete host '%s'? (y/n)", entry.Host)
	return detailPanelStyle.Width(m.width - 4).Height(10).Render(
		titleStyle.Render("Confirm Delete") + "\n\n" +
			warningStyle.Render(msg) + "\n\n" +
			helpStyle.Render("y: confirm | n/Esc: cancel"),
	)
}

func (m *Model) renderClearVisitsConfirm() string {
	msg := "Clear all visit counts? This will reset the visit history for all hosts."
	return detailPanelStyle.Width(m.width - 4).Height(10).Render(
		titleStyle.Render("Clear Visit Counts") + "\n\n" +
			warningStyle.Render(msg) + "\n\n" +
			helpStyle.Render("y: confirm | n/Esc: cancel"),
	)
}

// Helper functions
func getHostNames(entries []*sshconfig.HostEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Host
	}
	return names
}

func sortEntriesByHosts(entries []*sshconfig.HostEntry, sortedHosts []string) []*sshconfig.HostEntry {
	entryMap := make(map[string]*sshconfig.HostEntry)
	for _, entry := range entries {
		entryMap[entry.Host] = entry
	}

	sorted := make([]*sshconfig.HostEntry, 0, len(entries))
	for _, host := range sortedHosts {
		if entry, ok := entryMap[host]; ok {
			sorted = append(sorted, entry)
		}
	}

	// Add any entries not in sortedHosts (shouldn't happen, but safety check)
	for _, entry := range entries {
		found := false
		for _, host := range sortedHosts {
			if entry.Host == host {
				found = true
				break
			}
		}
		if !found {
			sorted = append(sorted, entry)
		}
	}

	return sorted
}
