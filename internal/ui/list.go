package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicklasos/gosshit/internal/sshconfig"
)

// formatTagBadge returns a styled badge for a tag
func formatTagBadge(tag string) string {
	tagLower := strings.ToLower(tag)
	switch tagLower {
	case "prod":
		return tagProdStyle.Render("[" + tag + "]")
	case "dev":
		return tagDevStyle.Render("[" + tag + "]")
	case "stage":
		return tagStageStyle.Render("[" + tag + "]")
	default:
		return tagDefaultStyle.Render("[" + tag + "]")
	}
}

// ListModel represents the left panel list view
type ListModel struct {
	entries     []*sshconfig.HostEntry
	filtered    []*sshconfig.HostEntry
	selected    int
	searchTerm  string
	width       int
	height      int
	visitCounts map[string]int // host -> visit count
}

// NewListModel creates a new list model
func NewListModel(entries []*sshconfig.HostEntry, visitCounts map[string]int) *ListModel {
	return &ListModel{
		entries:     entries,
		filtered:    entries,
		selected:    0,
		visitCounts: visitCounts,
	}
}

// Init initializes the list model
func (m *ListModel) Init() tea.Cmd {
	return nil
}

// Update handles updates to the list model
func (m *ListModel) Update(msg tea.Msg) (*ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "j", "down":
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
		}
	}
	return m, nil
}

// SetEntries updates the entries list
func (m *ListModel) SetEntries(entries []*sshconfig.HostEntry) {
	m.entries = entries
	m.ApplyFilter()
}

// SetVisitCounts updates the visit counts
func (m *ListModel) SetVisitCounts(counts map[string]int) {
	m.visitCounts = counts
}

// ApplyFilter applies the current search filter
func (m *ListModel) ApplyFilter() {
	if m.searchTerm == "" {
		m.filtered = m.entries
		m.selected = 0
		return
	}

	var filtered []*sshconfig.HostEntry
	term := strings.ToLower(m.searchTerm)
	for _, entry := range m.entries {
		// Check host, hostname, user, description
		if strings.Contains(strings.ToLower(entry.Host), term) ||
			strings.Contains(strings.ToLower(entry.HostName), term) ||
			strings.Contains(strings.ToLower(entry.User), term) ||
			strings.Contains(strings.ToLower(entry.Description), term) {
			filtered = append(filtered, entry)
			continue
		}
		// Check tags
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), term) {
				filtered = append(filtered, entry)
				break
			}
		}
	}

	m.filtered = filtered
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}
}

// SetSearchTerm sets the search term and applies the filter
func (m *ListModel) SetSearchTerm(term string) {
	m.searchTerm = term
	m.ApplyFilter()
}

// GetSelected returns the currently selected entry
func (m *ListModel) GetSelected() *sshconfig.HostEntry {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	return m.filtered[m.selected]
}

// SetSelected sets the selected index
func (m *ListModel) SetSelected(index int) {
	if index >= 0 && index < len(m.filtered) {
		m.selected = index
	} else if index < 0 {
		m.selected = 0
	} else if index >= len(m.filtered) && len(m.filtered) > 0 {
		m.selected = len(m.filtered) - 1
	}
}

// GetSelectedIndex returns the currently selected index
func (m *ListModel) GetSelectedIndex() int {
	return m.selected
}

// SetSize sets the size of the list view
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the list view
func (m *ListModel) View() string {
	if len(m.filtered) == 0 {
		return listPanelStyle.Width(m.width).Height(m.height).Render(
			titleStyle.Render("SSH Hosts") + "\n\n" +
				"No hosts found",
		)
	}

	var lines []string
	lines = append(lines, titleStyle.Render("SSH Hosts"))

	// Account for panel padding (1 top + 1 bottom) and title (1 line + margin)
	// Each entry can be 2-3 lines (2 lines normally, 3 when tags wrap)
	availableHeight := m.height - 2 - 2 // panel padding top/bottom
	titleHeight := 2                    // title + margin
	availableForEntries := availableHeight - titleHeight
	// Use a conservative estimate: assume 2.5 lines per entry on average
	visibleEntries := max(1, availableForEntries/3)

	start := max(0, m.selected-visibleEntries/2)
	end := min(len(m.filtered), start+visibleEntries*2) // Allow more entries to account for variable heights

	entryLinesCount := 0
	actualEnd := start
	
	for i := start; i < end && entryLinesCount < availableForEntries; i++ {
		entry := m.filtered[i]
		entryLines := m.formatEntry(entry, i == m.selected)
		splitLines := strings.Split(entryLines, "\n")
		if entryLinesCount+len(splitLines) > availableForEntries {
			break
		}
		for _, line := range splitLines {
			lines = append(lines, line)
			entryLinesCount++
		}
		actualEnd = i + 1
	}

	// Show scroll indicators
	hasMoreAbove := start > 0
	hasMoreBelow := actualEnd < len(m.filtered)
	
	if hasMoreAbove {
		lines = append([]string{lines[0], "..."}, lines[1:]...)
	}
	if hasMoreBelow {
		lines = append(lines, "...")
	}

	// Fill remaining space to ensure consistent height and proper border rendering
	for len(lines) < availableHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return listPanelStyle.Width(m.width).Height(m.height).Render(content)
}

// formatEntry formats a single entry for display
func (m *ListModel) formatEntry(entry *sshconfig.HostEntry, selected bool) string {
	// Format: Host name (main line)
	//         IP/hostname (smaller text below)

	hostname := entry.HostName
	if hostname == "" {
		hostname = entry.Host
	}

	// Add port if present
	if entry.Port != "" && entry.Port != "22" {
		hostname += ":" + entry.Port
	}

	// Main line: Host alias with tags
	hostAlias := entry.Host
	// Add tag badges
	var tagBadges []string
	for _, tag := range entry.Tags {
		tagBadges = append(tagBadges, formatTagBadge(tag))
	}

	mainLine := hostAlias
	var tagLine string
	if len(tagBadges) > 0 {
		if len(tagBadges) > 2 {
			// More than 2 tags: put all tags on a new line
			tagLine = "  " + strings.Join(tagBadges, " ")
		} else {
			// 2 or fewer tags: all on main line
			mainLine += " " + strings.Join(tagBadges, " ")
		}
	}
	if selected {
		mainLine = "▶ " + mainLine
	} else {
		mainLine = "  " + mainLine
	}
	// tagLine already has "  " prefix for alignment

	// Second line: IP/hostname in smaller, subtler text (indented to match main line)
	subLine := "  " + hostname

	// Style based on selection
	var linesToJoin []string
	if selected {
		linesToJoin = append(linesToJoin, listItemSelectedStyle.Render(mainLine))
		if tagLine != "" {
			// Style tag line with same border styling
			linesToJoin = append(linesToJoin, listItemSelectedStyle.Copy().
				Foreground(accentColor).
				Render(tagLine))
		}
		// IP line needs same border styling but different text color
		linesToJoin = append(linesToJoin, listItemSelectedStyle.Copy().
			Foreground(accentColor). // Use accent color for IP when selected
			Render(subLine))
	} else {
		linesToJoin = append(linesToJoin, listItemStyle.Render(mainLine))
		if tagLine != "" {
			linesToJoin = append(linesToJoin, listItemStyle.Copy().
				Foreground(subtleColor).
				Render(tagLine))
		}
		linesToJoin = append(linesToJoin, listItemStyle.Copy().
			Foreground(subtleColor).
			Render(subLine))
	}

	return lipgloss.JoinVertical(lipgloss.Left, linesToJoin...)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
