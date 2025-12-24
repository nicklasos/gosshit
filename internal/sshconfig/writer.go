package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteConfig writes the SSH config file with the given entries and standalone comments
func WriteConfig(path string, entries []*HostEntry, standaloneComments []string) error {
	// Expand tilde in path
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// Ensure .ssh directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write standalone comments at the top
	if len(standaloneComments) > 0 {
		for _, comment := range standaloneComments {
			if _, err := file.WriteString(comment + "\n"); err != nil {
				return fmt.Errorf("failed to write comment: %w", err)
			}
		}
		if len(entries) > 0 {
			if _, err := file.WriteString("\n"); err != nil {
				return fmt.Errorf("failed to write newline: %w", err)
			}
		}
	}

	// Write entries
	for i, entry := range entries {
		if err := writeEntry(file, entry); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
		// Add blank line between entries (except after the last one)
		if i < len(entries)-1 {
			if _, err := file.WriteString("\n"); err != nil {
				return fmt.Errorf("failed to write newline: %w", err)
			}
		}
	}

	return nil
}

// writeEntry writes a single host entry to the file
func writeEntry(file *os.File, entry *HostEntry) error {
	// If we have raw lines, try to preserve them (with updates)
	if len(entry.RawLines) > 0 {
		// Write description comment if we have one
		if entry.Description != "" {
			hasDesc := false
			for _, line := range entry.RawLines {
				if strings.Contains(line, "# Description:") {
					hasDesc = true
					break
				}
			}
			if !hasDesc {
				if _, err := file.WriteString("# Description: " + entry.Description + "\n"); err != nil {
					return err
				}
			}
		}

		// Detect indentation style from the first non-empty, non-comment, non-Host line
		indent := "    " // default to 4 spaces
		for _, l := range entry.RawLines {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(strings.ToLower(trimmed), "host ") {
				// Get the leading whitespace (preserves tabs/spaces)
				leading := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
				if len(leading) > 0 {
					indent = leading
					break
				}
			}
		}

		// Track which directives we've written
		writtenHostname := false
		writtenUser := false
		writtenPort := false

		// Write raw lines, updating values as needed
		for _, line := range entry.RawLines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				// Preserve comments and empty lines
				if _, err := file.WriteString(line + "\n"); err != nil {
					return err
				}
				continue
			}

			parts := strings.Fields(trimmed)
			if len(parts) < 2 {
				if _, err := file.WriteString(line + "\n"); err != nil {
					return err
				}
				continue
			}

			directive := strings.ToLower(parts[0])

			// Get original indentation and directive name from this line
			originalIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			originalDirective := parts[0] // Preserve original case

			// Update directives if they've changed, preserving original indentation and case
			switch directive {
			case "host":
				if _, err := file.WriteString("Host " + entry.Host + "\n"); err != nil {
					return err
				}
			case "hostname":
				writtenHostname = true
				newValue := strings.Join(parts[1:], " ")
				if newValue != entry.HostName {
					// Value changed, update it but preserve indentation and directive case
					if _, err := file.WriteString(originalIndent + originalDirective + " " + entry.HostName + "\n"); err != nil {
						return err
					}
				} else {
					// Value unchanged, write original line exactly as-is
					if _, err := file.WriteString(line + "\n"); err != nil {
						return err
					}
				}
			case "user":
				writtenUser = true
				newValue := strings.Join(parts[1:], " ")
				if entry.User != "" {
					if newValue != entry.User {
						// Value changed, update it but preserve indentation and directive case
						if _, err := file.WriteString(originalIndent + originalDirective + " " + entry.User + "\n"); err != nil {
							return err
						}
					} else {
						// Value unchanged, write original line exactly as-is
						if _, err := file.WriteString(line + "\n"); err != nil {
							return err
						}
					}
				} else {
					// User was removed, skip this line
					continue
				}
			case "port":
				writtenPort = true
				newValue := strings.Join(parts[1:], " ")
				if entry.Port != "" {
					if newValue != entry.Port {
						// Value changed, update it but preserve indentation and directive case
						if _, err := file.WriteString(originalIndent + originalDirective + " " + entry.Port + "\n"); err != nil {
							return err
						}
					} else {
						// Value unchanged, write original line exactly as-is
						if _, err := file.WriteString(line + "\n"); err != nil {
							return err
						}
					}
				} else {
					// Port was removed, skip this line
					continue
				}
			default:
				// Preserve other directives as-is
				if _, err := file.WriteString(line + "\n"); err != nil {
					return err
				}
			}
		}

		// Ensure required directives are present (only add if missing)
		if !writtenHostname && entry.HostName != "" {
			if _, err := file.WriteString(indent + "HostName " + entry.HostName + "\n"); err != nil {
				return err
			}
		}
		if !writtenUser && entry.User != "" {
			if _, err := file.WriteString(indent + "User " + entry.User + "\n"); err != nil {
				return err
			}
		}
		if !writtenPort && entry.Port != "" {
			if _, err := file.WriteString(indent + "Port " + entry.Port + "\n"); err != nil {
				return err
			}
		}

		return nil
	}

	// Write new entry from scratch
	if entry.Description != "" {
		if _, err := file.WriteString("# Description: " + entry.Description + "\n"); err != nil {
			return err
		}
	}

	if _, err := file.WriteString("Host " + entry.Host + "\n"); err != nil {
		return err
	}

	if entry.HostName != "" {
		if _, err := file.WriteString("    HostName " + entry.HostName + "\n"); err != nil {
			return err
		}
	}

	if entry.User != "" {
		if _, err := file.WriteString("    User " + entry.User + "\n"); err != nil {
			return err
		}
	}

	if entry.Port != "" {
		if _, err := file.WriteString("    Port " + entry.Port + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// AddEntry adds a new entry to the config file
func AddEntry(path string, entry *HostEntry) error {
	entries, standaloneComments, err := ParseConfig(path)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	entries = append(entries, entry)
	return WriteConfig(path, entries, standaloneComments)
}

// UpdateEntry updates an existing entry in the config file
func UpdateEntry(path string, oldHost string, newEntry *HostEntry) error {
	entries, standaloneComments, err := ParseConfig(path)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	for i, entry := range entries {
		if entry.Host == oldHost {
			entries[i] = newEntry
			break
		}
	}

	return WriteConfig(path, entries, standaloneComments)
}

// DeleteEntry removes an entry from the config file
func DeleteEntry(path string, host string) error {
	entries, standaloneComments, err := ParseConfig(path)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	var newEntries []*HostEntry
	for _, entry := range entries {
		if entry.Host != host {
			newEntries = append(newEntries, entry)
		}
	}

	return WriteConfig(path, newEntries, standaloneComments)
}
