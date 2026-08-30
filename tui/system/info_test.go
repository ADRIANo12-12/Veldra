// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Tests for the real system-info gatherer.

package system

import (
	"strings"
	"testing"
)

func TestCurrentInfoFieldsPresent(t *testing.T) {
	info := Current()
	if info.Version == "" {
		t.Error("Veldra version should not be empty")
	}
	if info.Arch == "" {
		t.Error("arch should not be empty")
	}
	if info.Hostname == "" {
		t.Error("hostname should not be empty")
	}
	if info.Uptime == "" {
		t.Error("uptime should not be empty")
	}
	if info.Kernel == "" {
		t.Error("kernel version should not be empty")
	}
	if info.CurrentUser == "" {
		t.Error("current user should not be empty")
	}
}

func TestMemoryParsing(t *testing.T) {
	total, avail, free := memory()
	if total == 0 {
		t.Error("memory() should report a total")
	}
	if avail > total {
		t.Errorf("available memory %d should not exceed total %d", avail, total)
	}
	if free > total {
		t.Errorf("free memory %d should not exceed total %d", free, total)
	}
}

func TestNetInterfacesReal(t *testing.T) {
	ifaces, err := netIfaces()
	if err != nil {
		t.Fatalf("netIfaces: %v", err)
	}
	found := false
	for _, n := range ifaces {
		if n == "lo" {
			found = true
		}
	}
	if !found {
		t.Error("loopback should always exist among netIfaces")
	}
}

func TestUptimeReal(t *testing.T) {
	u := uptime()
	if u == "unknown" {
		t.Error("uptime should be parseable on a real system")
	}
	if strings.Contains(u, "unknown") {
		t.Errorf("bogus uptime string: %q", u)
	}
}

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(86_400_000_000_000); got != "1d 0h 0m" {
		t.Errorf("expected 1d 0h 0m, got %q", got)
	}
	if got := formatDuration(3_600_000_000_000); got != "1h 0m" {
		t.Errorf("expected 1h 0m, got %q", got)
	}
}