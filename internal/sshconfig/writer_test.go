package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfig(t *testing.T) {
	tests := []struct {
		name               string
		entries            []*HostEntry
		standaloneComments []string
		wantContains       []string
	}{
		{
			name: "write single entry",
			entries: []*HostEntry{
				{
					Host:     "example",
					HostName: "example.com",
					User:     "root",
					Port:     "22",
				},
			},
			wantContains: []string{"Host example", "HostName example.com", "User root", "Port 22"},
		},
		{
			name: "write multiple entries",
			entries: []*HostEntry{
				{
					Host:     "example1",
					HostName: "example1.com",
					User:     "root",
				},
				{
					Host:     "example2",
					HostName: "example2.com",
					User:     "admin",
					Port:     "2222",
				},
			},
			wantContains: []string{"Host example1", "Host example2", "example1.com", "example2.com"},
		},
		{
			name: "write with IdentityFile",
			entries: []*HostEntry{
				{
					Host:         "github",
					HostName:     "github.com",
					User:         "git",
					IdentityFile: "~/.ssh/id_rsa_github",
				},
			},
			wantContains: []string{"Host github", "IdentityFile ~/.ssh/id_rsa_github"},
		},
		{
			name: "write with description",
			entries: []*HostEntry{
				{
					Host:        "prod",
					HostName:    "prod.example.com",
					Description: "Production server",
				},
			},
			wantContains: []string{"# Description: Production server", "Host prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config")

			err := WriteConfig(configPath, tt.entries, tt.standaloneComments)
			if err != nil {
				t.Fatalf("WriteConfig failed: %v", err)
			}

			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			contentStr := string(content)
			for _, want := range tt.wantContains {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Config should contain %q, got:\n%s", want, contentStr)
				}
			}
		})
	}
}

func TestAddEntry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create initial config
	initialEntry := &HostEntry{
		Host:     "existing",
		HostName: "existing.com",
		User:     "root",
	}
	err := WriteConfig(configPath, []*HostEntry{initialEntry}, nil)
	if err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Add new entry
	newEntry := &HostEntry{
		Host:     "new",
		HostName: "new.com",
		User:     "admin",
		Port:     "2222",
	}
	err = AddEntry(configPath, newEntry)
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	// Verify both entries exist
	entries, _, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	hosts := make(map[string]bool)
	for _, e := range entries {
		hosts[e.Host] = true
	}

	if !hosts["existing"] || !hosts["new"] {
		t.Error("Both 'existing' and 'new' hosts should be present")
	}
}

func TestUpdateEntry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create initial config
	initialEntry := &HostEntry{
		Host:     "example",
		HostName: "example.com",
		User:     "root",
		Port:     "22",
	}
	err := WriteConfig(configPath, []*HostEntry{initialEntry}, nil)
	if err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Update entry
	updatedEntry := &HostEntry{
		Host:     "example",
		HostName: "updated.example.com",
		User:     "admin",
		Port:     "2222",
	}
	err = UpdateEntry(configPath, "example", updatedEntry)
	if err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	// Verify update
	entries, _, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].HostName != "updated.example.com" {
		t.Errorf("HostName not updated: got %q, want %q", entries[0].HostName, "updated.example.com")
	}
	if entries[0].User != "admin" {
		t.Errorf("User not updated: got %q, want %q", entries[0].User, "admin")
	}
	if entries[0].Port != "2222" {
		t.Errorf("Port not updated: got %q, want %q", entries[0].Port, "2222")
	}
}

func TestDeleteEntry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create initial config with two entries
	entries := []*HostEntry{
		{
			Host:     "keep",
			HostName: "keep.com",
		},
		{
			Host:     "delete",
			HostName: "delete.com",
		},
	}
	err := WriteConfig(configPath, entries, nil)
	if err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Delete entry
	err = DeleteEntry(configPath, "delete")
	if err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}

	// Verify deletion
	remainingEntries, _, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(remainingEntries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(remainingEntries))
	}

	if remainingEntries[0].Host != "keep" {
		t.Errorf("Wrong entry remains: got %q, want %q", remainingEntries[0].Host, "keep")
	}
}

func TestPreserveFormatting(t *testing.T) {
	configContent := `Host example
	HostName example.com
	User root
	Port 22
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Write initial config with tabs
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Parse and write back
	entries, comments, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	err = WriteConfig(configPath, entries, comments)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Read and verify tabs are preserved
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "\t") {
		t.Error("Tabs should be preserved in formatting")
	}
}

func TestUpdateDescription(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create initial config with description
	initialEntry := &HostEntry{
		Host:        "example",
		HostName:    "example.com",
		User:        "root",
		Description: "Original description",
	}
	err := WriteConfig(configPath, []*HostEntry{initialEntry}, nil)
	if err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Parse it back
	entries, comments, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Description != "Original description" {
		t.Errorf("Initial description: got %q, want %q", entries[0].Description, "Original description")
	}

	// Update the description
	entries[0].Description = "Updated description"
	err = WriteConfig(configPath, entries, comments)
	if err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	// Verify the description was updated
	entries, _, err = ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig (second) failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry after update, got %d", len(entries))
	}

	if entries[0].Description != "Updated description" {
		t.Errorf("Updated description: got %q, want %q", entries[0].Description, "Updated description")
	}

	// Verify only one description comment exists
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	descCount := strings.Count(string(content), "# Description:")
	if descCount != 1 {
		t.Errorf("Expected 1 description comment, got %d", descCount)
	}

	if !strings.Contains(string(content), "# Description: Updated description") {
		t.Error("Config should contain updated description")
	}

	if strings.Contains(string(content), "Original description") {
		t.Error("Config should not contain old description")
	}
}
