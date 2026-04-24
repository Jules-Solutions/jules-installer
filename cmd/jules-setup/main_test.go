// Package main tests for the --tier flag parser.
package main

import (
	"testing"

	"github.com/Jules-Solutions/jules-installer/internal/config"
)

func TestResolveTierFlag(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		want    config.Tier
		wantErr bool
	}{
		// Empty flag → "no preference, show picker"
		{"empty is valid and empty", "", "", false},

		// Tier 1 aliases
		{"numeric 1", "1", config.TierFull, false},
		{"canonical tier1", "tier1", config.TierFull, false},
		{"descriptive full", "full", config.TierFull, false},
		{"uppercase FULL", "FULL", config.TierFull, false},
		{"mixed case Tier1 with spaces", "  Tier1  ", config.TierFull, false},

		// Tier 2 aliases
		{"numeric 2", "2", config.TierRemote, false},
		{"canonical tier2", "tier2", config.TierRemote, false},
		{"descriptive remote", "remote", config.TierRemote, false},
		{"uppercase REMOTE", "REMOTE", config.TierRemote, false},

		// Rejected values
		{"gibberish", "xyz", "", true},
		{"number 3", "3", "", true},
		{"tier3", "tier3", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTierFlag(tc.flag)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveTierFlag(%q) expected error, got nil (value=%q)", tc.flag, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveTierFlag(%q) unexpected error: %v", tc.flag, err)
			}
			if got != tc.want {
				t.Errorf("resolveTierFlag(%q) = %q, want %q", tc.flag, got, tc.want)
			}
		})
	}
}
