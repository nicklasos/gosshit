package sshconfig

// HostEntry represents a single SSH host configuration entry
type HostEntry struct {
	Host         string   // Host alias
	HostName     string   // HostName directive
	User         string   // User directive
	Port         string   // Port directive
	IdentityFile string   // IdentityFile directive
	Description  string   // Extracted from comment above Host entry
	Comment      string   // Original comment block
	RawLines     []string // Original lines for preservation
	StartLine    int      // Starting line number in original file
	EndLine      int      // Ending line number in original file
}

// IsValid checks if the host entry has the minimum required fields
// Host * entries are valid without HostName (they're global config blocks)
func (h *HostEntry) IsValid() bool {
	if h.Host == "" {
		return false
	}
	// Host * entries don't need HostName
	if h.Host == "*" {
		return true
	}
	// Regular entries need HostName
	return h.HostName != ""
}

// GetConnectionString returns the SSH connection string (user@hostname)
func (h *HostEntry) GetConnectionString() string {
	if h.User != "" {
		return h.User + "@" + h.HostName
	}
	return h.HostName
}

// GetSSHCommand returns the full SSH command string
func (h *HostEntry) GetSSHCommand() string {
	cmd := "ssh"
	if h.Port != "" {
		cmd += " -p " + h.Port
	}
	cmd += " " + h.GetConnectionString()
	return cmd
}
