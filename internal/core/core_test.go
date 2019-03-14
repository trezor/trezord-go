package core

import (
	"testing"

	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
)

func TestEnumerateEntriesSort(t *testing.T) {
	entries := []types.EnumerateEntry{
		{Path: "b"},
		{Path: "a"},
		{Path: "ab"},
	}
	EnumerateEntries(entries).Sort()
	if entries[0].Path != "a" || entries[1].Path != "ab" {
		t.Errorf("EnumerateEntries(entries).Sort() did not work well. The result: %v", entries)
	}
}
