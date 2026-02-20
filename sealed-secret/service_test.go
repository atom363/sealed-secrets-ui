package sealedsecret

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToStringSet(t *testing.T) {
	got := toStringSet([]string{
		"argocd.argoproj.io/sync-wave",
		"  argocd.argoproj.io/sync-wave  ",
		" custom.example/key ",
		"",
	})

	assert.Equal(t, map[string]struct{}{
		"argocd.argoproj.io/sync-wave": {},
		"custom.example/key":           {},
	}, got)
}

func TestGetScopeAnnotations(t *testing.T) {
	tcs := []struct {
		name  string
		scope string
		want  map[string]string
	}{
		{
			name:  "cluster",
			scope: "cluster",
			want:  map[string]string{"sealedsecrets.bitnami.com/cluster-wide": "true"},
		},
		{
			name:  "namespace",
			scope: "namespace",
			want:  map[string]string{"sealedsecrets.bitnami.com/namespace-wide": "true"},
		},
		{
			name:  "strict",
			scope: "strict",
			want:  map[string]string{},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := getScopeAnnotations(tc.scope)
			assert.Equal(t, tc.want, got)
		})
	}
}
