// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Package system gathers real information from the running Linux system for
// the Veldra TUI: kernel, hostname, uptime, CPU, memory, users, shell, and
// network interfaces. Everything is read from /proc, /sys, /etc, or system
// utilities at runtime — never hardcoded.

package system

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Info is a snapshot of real system facts.
type Info struct {
	Version      string // Veldra version
	Channel      string // release channel
	Status       string // build status
	Arch         string
	Kernel       string // uname release
	KernelFull   string // uname -r full
	Hostname     string
	Uptime       string
	CPUModel     string
	CPUCores     int
	MemTotal     uint64
	MemAvailable uint64
	MemFree      uint64
	CurrentUser  string
	Shell        string
	OSPretty     string
	OSIDLike     string
	Interfaces   []NetIface
	DistroID     string
}

// NetIface describes a single network interface.
type NetIface struct {
	Name      string
	State     string // up / down
	Addresses []string
	HasIP     bool
}

var (
	once     sync.Once
	cachedVd string
)

// Overridable via -ldflags "-X veldra/tui/system.Version=0.0.1-pre-alpha".
// The build injects the real version from config/veldra.conf at build time.
var (
	Version     = "0.0.1-pre-alpha"
	Channel     = "UNSTABLE"
	BuildStatus = "BOOTABLE"
	ReleaseArch = "x86_64"
)

func veldraVersion() string {
	if Version != "" {
		return Version
	}
	once.Do(func() {
		cachedVd = "0.0.1-pre-alpha"
		if b, err := os.ReadFile("/etc/os-release"); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(line, "VERSION_ID=") {
					v := strings.Trim(line[len("VERSION_ID="):], "\"")
					if v != "" {
						cachedVd = v
					}
					break
				}
			}
		}
	})
	return cachedVd
}

// Current returns a populated Info snapshot.
func Current() Info {
	info := Info{
		Version:     veldraVersion(),
		Channel:     Channel,
		Status:      BuildStatus,
		Arch:        ReleaseArch,
		CurrentUser: currentUser(),
		Shell:       currentShell(),
	}
	if unameR, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		info.Kernel = strings.TrimSpace(string(unameR))
	}
	info.KernelFull = info.Kernel
	info.Hostname = hostname()
	info.Uptime = uptime()
	info.CPUModel = cpuModel()
	info.CPUCores = runtime.NumCPU()
	info.MemTotal, info.MemAvailable, info.MemFree = memory()
	info.OSPretty, info.OSIDLike, info.DistroID = osRelease()
	info.Interfaces = netInterfaces()
	return info
}

func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}

func currentShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		// report the final component; verify it exists to stay honest
		if len(s) > 0 && skippableShell(s) {
			base := s[strings.LastIndex(s, "/")+1:]
			return base
		}
		return s
	}
	return "bash"
}

func skippableShell(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}

func uptime() string {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return "unknown"
	}
	var sec float64
	fmt.Sscanf(fields[0], "%f", &sec)
	return formatDuration(time.Duration(int64(sec)) * time.Second)
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	days := total / 86400
	hours := (total % 86400) / 3600
	mins := (total % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func cpuModel() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "model name") {
			i := strings.Index(line, ":")
			if i >= 0 {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}
	return "unknown"
}

func memory() (total, available, free uint64) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	kv := map[string]*uint64{
		"MemTotal":     &total,
		"MemAvailable": &available,
		"MemFree":      &free,
	}
	for _, line := range strings.Split(string(b), "\n") {
		if i := strings.Index(line, ":"); i >= 0 {
			key := line[:i]
			if p, ok := kv[key]; ok {
				val := strings.Fields(line[i+1:])
				if len(val) > 0 {
					fmt.Sscanf(val[0], "%d", p)
				}
			}
		}
	}
	return
}

func osRelease() (pretty, idLike, id string) {
	kv := map[string]string{}
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", "", ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if i := strings.Index(line, "="); i >= 0 {
			key := line[:i]
			val := strings.Trim(line[i+1:], "\"")
			kv[key] = val
		}
	}
	return kv["PRETTY_NAME"], kv["ID_LIKE"], kv["ID"]
}

func netInterfaces() []NetIface {
	var out []NetIface
	ifaces, err := netIfaces()
	if err != nil {
		return out
	}
	for _, name := range ifaces {
		if name == "lo" {
			continue
		}
		ni := NetIface{Name: name}
		addr := interfaceAddrs(name)
		ni.Addresses = addr
		ni.HasIP = len(addr) > 0
		ni.State = "down"
		if operational(name) {
			ni.State = "up"
		}
		out = append(out, ni)
	}
	return out
}

// netIfaces reads interface names from /sys/class/net.
func netIfaces() ([]string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func interfaceAddrs(name string) []string {
	var out []string
	b, err := exec.Command("ip", "-4", "-o", "addr", "show", name).Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		if i := strings.Index(line, "inet "); i >= 0 {
			rest := line[i+len("inet "):]
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				out = append(out, fields[0])
			}
		}
	}
	return out
}

func operational(name string) bool {
	b, err := os.ReadFile("/sys/class/net/" + name + "/operstate")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b)) == "up"
}
