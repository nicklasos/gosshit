package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	sshConfigPath = "~/.ssh/config"
)

// GetSSHConfigPath returns the expanded path to the SSH config file
func GetSSHConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return sshConfigPath
	}
	return filepath.Join(homeDir, ".ssh", "config")
}

// ParseConfig reads and parses the SSH config file, returning a list of HostEntry
func ParseConfig(path string) ([]*HostEntry, []string, error) {
	// Expand tilde in path
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = strings.Replace(path, "~", homeDir, 1)
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty list if file doesn't exist
			return []*HostEntry{}, []string{}, nil
		}
		return nil, nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var entries []*HostEntry
	var standaloneComments []string
	var currentEntry *HostEntry
	var commentBuffer []string
	var rawLines []string
	var currentHostLines []string
	lineNum := 0
	inHostBlock := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		rawLines = append(rawLines, line)

		trimmed := strings.TrimSpace(line)

		// Handle comments
		if strings.HasPrefix(trimmed, "#") {
			// Check if it's a description comment - this signals start of next host block
			if strings.HasPrefix(trimmed, "# Description:") || (strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "##") && inHostBlock) {
				// If we're in a host block and see a standalone comment (not indented),
				// it's likely the description for the NEXT host, so end current block
				if inHostBlock && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					// This comment is for the next host - end current host block
					if currentEntry != nil {
						currentEntry.RawLines = currentHostLines
						currentEntry.EndLine = lineNum - 1
						if currentEntry.IsValid() {
							entries = append(entries, currentEntry)
						}
					}
					inHostBlock = false
					currentEntry = nil
					currentHostLines = []string{}
				}
			}

			// Add comment to appropriate buffer
			if inHostBlock {
				currentHostLines = append(currentHostLines, line)
				if currentEntry != nil {
					currentEntry.Comment += line + "\n"
				}
			} else {
				commentBuffer = append(commentBuffer, line)
			}
			continue
		}

		// Handle empty lines
		if trimmed == "" {
			if inHostBlock {
				currentHostLines = append(currentHostLines, line)
				if currentEntry != nil {
					currentEntry.Comment += line + "\n"
				}
			} else {
				// Keep empty lines in comment buffer - they might be between comment and Host
				// Don't clear the buffer yet - wait for next non-empty, non-comment line
			}
			continue
		}

		// Parse directives
		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			if inHostBlock {
				currentHostLines = append(currentHostLines, line)
			}
			continue
		}

		directive := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		// Handle Host directive (start of new host block)
		if directive == "host" {
			// Save previous entry if it exists
			if currentEntry != nil && inHostBlock {
				currentEntry.RawLines = currentHostLines
				currentEntry.EndLine = lineNum - 1
				if currentEntry.IsValid() {
					entries = append(entries, currentEntry)
				}
			}

			// Start new entry
			inHostBlock = true
			currentHostLines = []string{}

			// Add comment buffer to new entry (excluding trailing empty lines)
			if len(commentBuffer) > 0 {
				for _, c := range commentBuffer {
					currentHostLines = append(currentHostLines, c)
				}
			}

			// Extract description from comment buffer
			desc := ""
			for _, c := range commentBuffer {
				trimmed := strings.TrimSpace(c)
				if trimmed == "" {
					// Skip empty lines
					continue
				}
				if strings.HasPrefix(trimmed, "# Description:") {
					// Explicit description format
					desc = strings.TrimPrefix(trimmed, "# Description:")
					desc = strings.TrimSpace(desc)
					break
				} else if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "##") {
					// Regular comment line - use as description if we don't have one yet
					if desc == "" {
						desc = strings.TrimPrefix(trimmed, "#")
						desc = strings.TrimSpace(desc)
						// Don't break - keep looking for explicit Description format
					}
				}
			}

			currentEntry = &HostEntry{
				Host:        value,
				Description: desc,
				StartLine:   lineNum,
				RawLines:    make([]string, 0),
			}

			// Add comment buffer to comment field
			if len(commentBuffer) > 0 {
				currentEntry.Comment = strings.Join(commentBuffer, "\n") + "\n"
			}

			commentBuffer = []string{}
			currentHostLines = append(currentHostLines, line)
			continue
		}

		// If we reach here with a non-Host directive outside a host block,
		// clear the comment buffer as it's not associated with a host
		if !inHostBlock && len(commentBuffer) > 0 {
			standaloneComments = append(standaloneComments, commentBuffer...)
			commentBuffer = []string{}
		}

		// Handle other directives within a host block
		if inHostBlock && currentEntry != nil {
			currentHostLines = append(currentHostLines, line)
			switch directive {
			case "hostname":
				currentEntry.HostName = value
			case "user":
				currentEntry.User = value
			case "port":
				currentEntry.Port = value
			case "identityfile":
				currentEntry.IdentityFile = value
			}
		} else {
			// Directive outside host block - treat as standalone
			if len(commentBuffer) > 0 {
				standaloneComments = append(standaloneComments, commentBuffer...)
				commentBuffer = []string{}
			}
		}
	}

	// Save last entry
	if currentEntry != nil && inHostBlock {
		currentEntry.RawLines = currentHostLines
		currentEntry.EndLine = lineNum
		if currentEntry.IsValid() {
			entries = append(entries, currentEntry)
		}
	}

	// Add any remaining standalone comments
	if len(commentBuffer) > 0 {
		standaloneComments = append(standaloneComments, commentBuffer...)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading config file: %w", err)
	}

	return entries, standaloneComments, nil
}
