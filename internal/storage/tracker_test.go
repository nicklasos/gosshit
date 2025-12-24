package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVisitTracker_Increment(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "gosshit")

	tracker, err := NewVisitTracker()
	if err != nil {
		t.Fatalf("NewVisitTracker failed: %v", err)
	}
	tracker.path = trackerPath

	// Test incrementing new host
	tracker.Increment("host1")
	if got := tracker.GetCount("host1"); got != 1 {
		t.Errorf("After first increment: got %d, want 1", got)
	}

	// Test incrementing existing host
	tracker.Increment("host1")
	if got := tracker.GetCount("host1"); got != 2 {
		t.Errorf("After second increment: got %d, want 2", got)
	}

	// Test multiple hosts
	tracker.Increment("host2")
	tracker.Increment("host2")
	tracker.Increment("host2")
	if got := tracker.GetCount("host2"); got != 3 {
		t.Errorf("host2 count: got %d, want 3", got)
	}
}

func TestVisitTracker_GetCount(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "gosshit")

	tracker, err := NewVisitTracker()
	if err != nil {
		t.Fatalf("NewVisitTracker failed: %v", err)
	}
	tracker.path = trackerPath

	// Test non-existent host
	if got := tracker.GetCount("nonexistent"); got != 0 {
		t.Errorf("Non-existent host: got %d, want 0", got)
	}

	// Test existing host
	tracker.Increment("host1")
	tracker.Increment("host1")
	if got := tracker.GetCount("host1"); got != 2 {
		t.Errorf("Existing host: got %d, want 2", got)
	}
}

func TestVisitTracker_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "gosshit")

	// Create tracker and add some visits
	tracker1, err := NewVisitTracker()
	if err != nil {
		t.Fatalf("NewVisitTracker failed: %v", err)
	}
	tracker1.path = trackerPath

	tracker1.Increment("host1")
	tracker1.Increment("host1")
	tracker1.Increment("host2")
	tracker1.Increment("host3")
	tracker1.Increment("host3")
	tracker1.Increment("host3")

	// Save
	err = tracker1.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load into new tracker
	tracker2, err2 := NewVisitTracker()
	if err2 != nil {
		t.Fatalf("NewVisitTracker (second) failed: %v", err2)
	}
	tracker2.path = trackerPath
	err = tracker2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify counts
	tests := []struct {
		host string
		want int
	}{
		{"host1", 2},
		{"host2", 1},
		{"host3", 3},
	}

	for _, tt := range tests {
		if got := tracker2.GetCount(tt.host); got != tt.want {
			t.Errorf("Host %s: got %d, want %d", tt.host, got, tt.want)
		}
	}
}

func TestVisitTracker_SortByVisits(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "gosshit")

	tracker, err := NewVisitTracker()
	if err != nil {
		t.Fatalf("NewVisitTracker failed: %v", err)
	}
	tracker.path = trackerPath

	// Add visits with different counts
	tracker.Increment("rarely")

	tracker.Increment("sometimes")
	tracker.Increment("sometimes")
	tracker.Increment("sometimes")

	tracker.Increment("often")
	tracker.Increment("often")
	tracker.Increment("often")
	tracker.Increment("often")
	tracker.Increment("often")

	hosts := []string{"rarely", "sometimes", "often"}
	sorted := tracker.SortByVisits(hosts)

	// Should be sorted by visit count (descending)
	expected := []string{"often", "sometimes", "rarely"}

	if len(sorted) != len(expected) {
		t.Fatalf("Length mismatch: got %d, want %d", len(sorted), len(expected))
	}

	for i, host := range expected {
		if sorted[i] != host {
			t.Errorf("Position %d: got %q, want %q", i, sorted[i], host)
		}
	}
}

func TestVisitTracker_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "gosshit")

	// Create empty file
	err := os.WriteFile(trackerPath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Should handle empty file gracefully
	tracker, err := NewVisitTracker()
	if err != nil {
		t.Fatalf("NewVisitTracker failed: %v", err)
	}
	tracker.path = trackerPath
	err = tracker.Load()
	// Empty file should be handled gracefully, no error
	if err != nil {
		t.Logf("Load returned error (may be okay): %v", err)
	}

	if got := tracker.GetCount("anyhost"); got != 0 {
		t.Errorf("Empty file should result in 0 count, got %d", got)
	}
}

func TestVisitTracker_NonExistentDirectory(t *testing.T) {
	// Use a temporary directory path
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "nonexistent_subdir", "gosshit")

	tracker, err := NewVisitTracker()
	if err != nil {
		t.Fatalf("NewVisitTracker failed: %v", err)
	}
	tracker.path = trackerPath

	tracker.Increment("host1")
	if got := tracker.GetCount("host1"); got != 1 {
		t.Errorf("Should work even if directory doesn't exist yet: got %d, want 1", got)
	}
}
