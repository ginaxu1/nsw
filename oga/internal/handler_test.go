package internal

import (
	"testing"
)

func TestStorageKeyRx(t *testing.T) {
	testCases := []struct {
		key   string
		valid bool
	}{
		{"abc", true},
		{"abc.pdf", true},
		{"abc/def.pdf", true},
		{"abc/def/ghi.pdf", true},
		{"archive.tar.gz", true},
		{"project.v1/report.pdf", true},
		{"archives/document-v1.2.pdf", true},
		{"..", false},
		{"../abc", false},
		{"abc/..", false},
		{"abc/../def", false},
		{"/abc", false},
		{"abc/", false},
		{"abc//def", false},
		{".abc", false},
		{"abc/.def", false},
	}

	for _, tc := range testCases {
		if storageKeyRx.MatchString(tc.key) != tc.valid {
			t.Errorf("expected %t for key %q, got %t", tc.valid, tc.key, !tc.valid)
		}
	}
}
