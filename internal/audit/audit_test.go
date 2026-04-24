// Package audit tests for the tier-aware severity + install filters.
package audit

import "testing"

func TestCheck_StatusForTier(t *testing.T) {
	cases := []struct {
		name   string
		check  Check
		tier   string
		wanted string // expected effective status for the tier
	}{
		// Tier 1 — no demotion, raw status is returned.
		{"tier1 python fail stays fail", Check{Name: "Python", Status: StatusFail}, "tier1", StatusFail},
		{"tier1 docker fail stays fail", Check{Name: "Docker", Status: StatusFail}, "tier1", StatusFail},
		{"tier1 CC fail stays fail", Check{Name: "Claude Code", Status: StatusFail}, "tier1", StatusFail},

		// Tier 2 — non-critical fails demote to warn.
		{"tier2 python fail demotes to warn", Check{Name: "Python", Status: StatusFail}, "tier2", StatusWarn},
		{"tier2 docker fail demotes to warn", Check{Name: "Docker", Status: StatusFail}, "tier2", StatusWarn},
		{"tier2 node fail demotes to warn", Check{Name: "Node.js", Status: StatusFail}, "tier2", StatusWarn},
		{"tier2 editors fail demotes to warn", Check{Name: "Editors", Status: StatusFail}, "tier2", StatusWarn},

		// Tier 2 — critical items stay critical.
		{"tier2 CC fail stays fail", Check{Name: "Claude Code", Status: StatusFail}, "tier2", StatusFail},

		// Status != fail is never demoted.
		{"tier2 python pass stays pass", Check{Name: "Python", Status: StatusPass}, "tier2", StatusPass},
		{"tier2 python warn stays warn", Check{Name: "Python", Status: StatusWarn}, "tier2", StatusWarn},

		// Empty tier falls back to raw status.
		{"empty tier returns raw status", Check{Name: "Python", Status: StatusFail}, "", StatusFail},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.check.StatusForTier(tc.tier)
			if got != tc.wanted {
				t.Errorf("StatusForTier(%q) on %s/%s = %q, want %q",
					tc.tier, tc.check.Name, tc.check.Status, got, tc.wanted)
			}
		})
	}
}

func TestCheck_InstallableForTier(t *testing.T) {
	// Use Python since it has an installer registered.
	pythonFail := Check{Name: "Python", Status: StatusFail}
	ccFail := Check{Name: "Claude Code", Status: StatusFail}

	// Tier 1: both should be installable.
	if !pythonFail.InstallableForTier("tier1") {
		t.Error("Tier 1 must offer to install Python on fail")
	}
	if !ccFail.InstallableForTier("tier1") {
		t.Error("Tier 1 must offer to install Claude Code on fail")
	}

	// Tier 2: CC yes, Python no.
	if pythonFail.InstallableForTier("tier2") {
		t.Error("Tier 2 must NOT offer to install Python")
	}
	if !ccFail.InstallableForTier("tier2") {
		t.Error("Tier 2 must still offer to install Claude Code (it's critical)")
	}

	// Passing check is never installable regardless of tier.
	pythonPass := Check{Name: "Python", Status: StatusPass}
	if pythonPass.InstallableForTier("tier1") {
		t.Error("Passing check is never installable")
	}
}

func TestCountInstallableForTier(t *testing.T) {
	checks := []Check{
		{Name: "Python", Status: StatusFail},       // fail on both tiers' detector, but Tier 2 skips it
		{Name: "Docker", Status: StatusFail},       // same
		{Name: "Claude Code", Status: StatusFail},  // critical on both tiers
		{Name: "Git", Status: StatusFail},          // fail on both, but Tier 2 skips it
		{Name: "Platform", Status: StatusPass},     // pass — not installable regardless
	}

	if got := CountInstallableForTier(checks, "tier1"); got != 4 {
		t.Errorf("Tier 1 installable count = %d, want 4", got)
	}
	if got := CountInstallableForTier(checks, "tier2"); got != 1 {
		t.Errorf("Tier 2 installable count = %d, want 1 (only Claude Code)", got)
	}
}
