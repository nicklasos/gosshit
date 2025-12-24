package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	trackerFileName = ".gosshit"
)

// GetTrackerPath returns the path to the visit tracker file
func GetTrackerPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, trackerFileName), nil
}

// VisitTracker manages visit counts for SSH hosts
type VisitTracker struct {
	counts map[string]int
	path   string
}

// NewVisitTracker creates a new VisitTracker and loads existing data
func NewVisitTracker() (*VisitTracker, error) {
	path, err := GetTrackerPath()
	if err != nil {
		return nil, err
	}

	tracker := &VisitTracker{
		counts: make(map[string]int),
		path:   path,
	}

	if err := tracker.Load(); err != nil {
		return nil, err
	}

	return tracker, nil
}

// Load reads the tracker file and loads visit counts into memory
func (vt *VisitTracker) Load() error {
	file, err := os.Open(vt.path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's okay
			return nil
		}
		return fmt.Errorf("failed to open tracker file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		host := strings.TrimSpace(parts[0])
		count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}

		vt.counts[host] = count
	}

	return scanner.Err()
}

// Save writes the visit counts to the tracker file
func (vt *VisitTracker) Save() error {
	file, err := os.Create(vt.path)
	if err != nil {
		return fmt.Errorf("failed to create tracker file: %w", err)
	}
	defer file.Close()

	// Sort by count (descending) for consistent output
	type hostCount struct {
		host  string
		count int
	}

	var entries []hostCount
	for host, count := range vt.counts {
		entries = append(entries, hostCount{host, count})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count == entries[j].count {
			return entries[i].host < entries[j].host
		}
		return entries[i].count > entries[j].count
	})

	for _, entry := range entries {
		if _, err := fmt.Fprintf(file, "%s:%d\n", entry.host, entry.count); err != nil {
			return fmt.Errorf("failed to write tracker entry: %w", err)
		}
	}

	return nil
}

// Increment increments the visit count for a host
func (vt *VisitTracker) Increment(host string) {
	vt.counts[host]++
}

// GetCount returns the visit count for a host (0 if not found)
func (vt *VisitTracker) GetCount(host string) int {
	return vt.counts[host]
}

// SortByVisits sorts a slice of host names by visit count (descending)
func (vt *VisitTracker) SortByVisits(hosts []string) []string {
	type hostWithCount struct {
		host  string
		count int
	}

	var hostsWithCounts []hostWithCount
	for _, host := range hosts {
		hostsWithCounts = append(hostsWithCounts, hostWithCount{
			host:  host,
			count: vt.GetCount(host),
		})
	}

	sort.Slice(hostsWithCounts, func(i, j int) bool {
		if hostsWithCounts[i].count == hostsWithCounts[j].count {
			return hostsWithCounts[i].host < hostsWithCounts[j].host
		}
		return hostsWithCounts[i].count > hostsWithCounts[j].count
	})

	result := make([]string, len(hostsWithCounts))
	for i, hwc := range hostsWithCounts {
		result[i] = hwc.host
	}

	return result
}
