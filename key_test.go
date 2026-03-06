package virefs

import (
	"errors"
	"testing"
)

func TestCleanKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
		isErr bool
	}{
		{"", "", false},
		{"/", "", false},
		{"a/b/c", "a/b/c", false},
		{"/a/b/c/", "a/b/c", false},
		{"a//b///c", "a/b/c", false},
		{"a/./b", "a/b", false},
		{"..", "", true},
		{"a/../../etc", "", true},
		{"a/../b", "b", false},
	}
	for _, tt := range tests {
		got, err := CleanKey(tt.input)
		if tt.isErr {
			if err == nil {
				t.Errorf("CleanKey(%q) expected error, got %q", tt.input, got)
			} else if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("CleanKey(%q) error should wrap ErrInvalidKey, got %v", tt.input, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("CleanKey(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("CleanKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
