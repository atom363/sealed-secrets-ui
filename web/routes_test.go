package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCSV(t *testing.T) {
	tcs := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  []string{},
		},
		{
			name:  "single value",
			input: "argocd.argoproj.io/sync-wave",
			want:  []string{"argocd.argoproj.io/sync-wave"},
		},
		{
			name:  "trims and skips empty values",
			input: " argocd.argoproj.io/sync-wave , , custom.example/key ",
			want:  []string{"argocd.argoproj.io/sync-wave", "custom.example/key"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCSV(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
