package sshconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		wantEntries   int
		wantHosts     []string
		wantHostNames []string
		wantComments  int
		expectError   bool
	}{
		{
			name: "simple config",
			configContent: `Host example
    HostName example.com
    User root
    Port 22

Host test
    HostName test.com
    User admin
`,
			wantEntries:   2,
			wantHosts:     []string{"example", "test"},
			wantHostNames: []string{"example.com", "test.com"},
			expectError:   false,
		},
		{
			name: "config with Host *",
			configContent: `Host *
    UseKeychain yes
    AddKeysToAgent yes

Host example
    HostName example.com
    User root
`,
			wantEntries:   2,
			wantHosts:     []string{"*", "example"},
			wantHostNames: []string{"", "example.com"},
			expectError:   false,
		},
		{
			name: "config with comments",
			configContent: `# Description: Production server
Host prod
    HostName prod.example.com
    User deploy

# This is a comment
# Another comment
Host staging
    HostName staging.example.com
`,
			wantEntries:   2,
			wantHosts:     []string{"prod", "staging"},
			wantHostNames: []string{"prod.example.com", "staging.example.com"},
			expectError:   false,
		},
		{
			name: "config with IdentityFile",
			configContent: `Host github
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_rsa_github

Host gitlab
    HostName gitlab.com
    User git
    IdentityFile ~/.ssh/id_ed25519
`,
			wantEntries:   2,
			wantHosts:     []string{"github", "gitlab"},
			wantHostNames: []string{"github.com", "gitlab.com"},
			expectError:   false,
		},
		{
			name:          "config with tabs",
			configContent: "Host example\n\tHostName example.com\n\tUser root\n\tPort 22\n",
			wantEntries:   1,
			wantHosts:     []string{"example"},
			wantHostNames: []string{"example.com"},
			expectError:   false,
		},
		{
			name:          "empty config",
			configContent: "",
			wantEntries:   0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config")

			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}

			entries, comments, err := ParseConfig(configPath)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(entries) != tt.wantEntries {
				t.Errorf("Got %d entries, want %d", len(entries), tt.wantEntries)
			}

			for i, host := range tt.wantHosts {
				if i >= len(entries) {
					break
				}
				if entries[i].Host != host {
					t.Errorf("Entry %d: got host %q, want %q", i, entries[i].Host, host)
				}
			}

			for i, hostname := range tt.wantHostNames {
				if i >= len(entries) {
					break
				}
				if entries[i].HostName != hostname {
					t.Errorf("Entry %d: got hostname %q, want %q", i, entries[i].HostName, hostname)
				}
			}

			if tt.wantComments > 0 && len(comments) != tt.wantComments {
				t.Errorf("Got %d comments, want %d", len(comments), tt.wantComments)
			}
		})
	}
}

func TestParseConfig_NonExistentFile(t *testing.T) {
	entries, comments, err := ParseConfig("/nonexistent/path/config")
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(comments))
	}
}

func TestParseConfig_PreservesRawLines(t *testing.T) {
	configContent := `Host example
    HostName example.com
    User root
    Port 22
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	entries, _, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries[0].RawLines) == 0 {
		t.Error("RawLines should not be empty")
	}

	// Check that RawLines contains the original content
	hasHost := false
	for _, line := range entries[0].RawLines {
		if line == "Host example" {
			hasHost = true
			break
		}
	}
	if !hasHost {
		t.Error("RawLines should contain 'Host example'")
	}
}
