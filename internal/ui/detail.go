package ui

import (
	"fmt"
	"strings"

	"github.com/nicklasos/gosshit/internal/sshconfig"
)

// DetailModel represents the right panel detail view
type DetailModel struct {
	entry      *sshconfig.HostEntry
	visitCount int
	width      int
	height     int
}

// NewDetailModel creates a new detail model
func NewDetailModel() *DetailModel {
	return &DetailModel{}
}

// SetEntry sets the entry to display
func (m *DetailModel) SetEntry(entry *sshconfig.HostEntry) {
	m.entry = entry
}

// SetVisitCount sets the visit count for the current entry
func (m *DetailModel) SetVisitCount(count int) {
	m.visitCount = count
}

// SetSize sets the size of the detail view
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the detail view
func (m *DetailModel) View() string {
	if m.entry == nil {
		return detailPanelStyle.Width(m.width).Height(m.height).Render(
			titleStyle.Render("Host Details") + "\n\n" +
				"No host selected",
		)
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Host Details"))

	// Description
	if m.entry.Description != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Description:"))
		lines = append(lines, valueStyle.Render(m.entry.Description))
	}

	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Host:"))
	lines = append(lines, valueStyle.Render(m.entry.Host))

	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("HostName:"))
	if m.entry.HostName != "" {
		lines = append(lines, valueStyle.Render(m.entry.HostName))
	} else {
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("(not set)"))
	}

	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("User:"))
	if m.entry.User != "" {
		lines = append(lines, valueStyle.Render(m.entry.User))
	} else {
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("(not set)"))
	}

	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Port:"))
	if m.entry.Port != "" {
		lines = append(lines, valueStyle.Render(m.entry.Port))
	} else {
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("(default: 22)"))
	}

	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("IdentityFile:"))
	if m.entry.IdentityFile != "" {
		lines = append(lines, valueStyle.Render(m.entry.IdentityFile))
	} else {
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("(not set)"))
	}

	// Connection string
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Connection:"))
	connStr := m.entry.GetConnectionString()
	if m.entry.Port != "" {
		connStr += ":" + m.entry.Port
	}
	lines = append(lines, valueStyle.Render(connStr))

	// SSH command
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("SSH Command:"))
	lines = append(lines, valueStyle.Foreground(accentColor).Render(m.entry.GetSSHCommand()))

	// Visit count
	if m.visitCount > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Visits:"))
		lines = append(lines, valueStyle.Render(fmt.Sprintf("%d", m.visitCount)))
	}

	content := strings.Join(lines, "\n")
	return detailPanelStyle.Width(m.width).Height(m.height).Render(content)
}
