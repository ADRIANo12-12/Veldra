// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Tests for the real /proc process collector.

package taskmanager

import (
	"testing"
)

func TestScanReturnsRealProcesses(t *testing.T) {
	c := NewCollector()
	procs := c.Refresh()
	if len(procs) == 0 {
		t.Fatal("expected at least one real process from /proc")
	}
	// the test binary is alive, so its own PID should be representable
	foundSelf := false
	for _, p := range procs {
		if p.Name != "" && p.PID > 0 {
			foundSelf = true
			break
		}
	}
	if !foundSelf {
		t.Error("should find at least one process with a name and PID")
	}
}

func TestStatParsesFields(t *testing.T) {
	c := &Collector{}
	procs := c.scan()
	for _, p := range procs {
		if p.PID <= 0 {
			t.Errorf("PID should be positive, got %d", p.PID)
		}
		if p.RSS > 1<<40 {
			t.Errorf("RSS %d looks implausible", p.RSS)
		}
	}
}

func TestMib(t *testing.T) {
	if got := Mib(1024); got != "1.0" {
		t.Errorf("Mib(1024) = %q, want 1.0", got)
	}
}