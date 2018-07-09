package core

import (
	"testing"
)

func TestEnumerateEntriesSort(t *testing.T) {
	entries := []EnumerateEntry{
		EnumerateEntry{Path: "b"},
		EnumerateEntry{Path: "a"},
		EnumerateEntry{Path: "ab"},
	}
	EnumerateEntries(entries).Sort()
	if entries[0].Path != "a" || entries[1].Path != "ab" {
		t.Errorf("EnumerateEntries(entries).Sort() did not work well. The result: %v", entries)
	}
}
