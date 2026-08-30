// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Package taskmanager reads real process information from /proc for the
// Veldra Task Manager application. No values are fabricated: PIDs, names,
// CPU time and memory RSS all come from the live kernel view.

package taskmanager

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Process is a single real process entry.
type Process struct {
	PID      int
	PPID     int
	Name     string
	CPU      float64 // percentage of one core since last sample
	RSS      uint64  // resident set size in KiB
	User     string
	VSZ      uint64 // virtual size in KiB
	State    string
	CPUseconds float64
}

// Collector samples /proc on each Refresh call and computes a delta-based
// CPU percentage between the previous sample and the current one.
type Collector struct {
	prev map[int]float64
	now  time.Time
}

// NewCollector returns a fresh process collector.
func NewCollector() *Collector {
	return &Collector{prev: map[int]float64{}, now: time.Now()}
}

// Refresh returns the current sorted list of processes.
func (c *Collector) Refresh() []Process {
	procs := c.scan()
	if len(procs) == 0 {
		return procs
	}
	return procs
}

func (c *Collector) scan() []Process {
	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	procs := make([]Process, 0, len(dirs))
	next := map[int]float64{}
	for _, d := range dirs {
		pid, err := strconv.Atoi(d.Name())
		if err != nil {
			continue
		}
		p := c.stat(pid)
		if p == nil {
			continue
		}
		prevCPU := c.prev[pid]
		delta := p.CPUseconds - prevCPU
		elapsed := time.Since(c.now).Seconds()
		if elapsed > 0 {
			p.CPU = delta / elapsed * 100.0
			if p.CPU < 0 {
				p.CPU = 0
			}
		}
		next[pid] = p.CPUseconds
		p.User = readUser(pid)
		procs = append(procs, *p)
	}
	c.prev = next
	c.now = time.Now()

	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPU == procs[j].CPU {
			return procs[i].PID < procs[j].PID
		}
		return procs[i].CPU > procs[j].CPU
	})
	return procs
}

func (c *Collector) stat(pid int) *Process {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return nil
	}
	s := string(stat)
	// find the last ')' to robustly split the comm field
	rparen := strings.LastIndex(s, ")")
	if rparen < 0 {
		return nil
	}
	comm := s[strings.Index(s, "(")+1 : rparen]
	rest := strings.Fields(s[rparen+2:])
	if len(rest) < 19 {
		return nil
	}
	p := &Process{PID: pid, Name: comm}
	// rest layout (0-based after comm): state=0 ppid=1 utime=11 stime=12 rss=21 vsize=22
	p.State = rest[0]
	if ppid, err := strconv.Atoi(rest[1]); err == nil {
		p.PPID = ppid
	}
	var utime, stime uint64
	if v, err := strconv.ParseUint(rest[11], 10, 64); err == nil {
		utime = v
	}
	if v, err := strconv.ParseUint(rest[12], 10, 64); err == nil {
		stime = v
	}
	hz := float64(100) // USER_HZ is 100 on x86_64
	p.CPUseconds = (float64(utime) + float64(stime)) / hz
	// rest[21] = rss pages, rest[22] = vsize bytes
	if len(rest) > 22 {
		if v, err := strconv.ParseUint(rest[21], 10, 64); err == nil {
			p.RSS = v * uint64(os.Getpagesize()/1024) // pages -> KiB
		}
		if v, err := strconv.ParseUint(rest[22], 10, 64); err == nil {
			p.VSZ = v / 1024
		}
	}
	return p
}

func readUser(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/loginuid", pid))
	if err != nil {
		return "?"
	}
	uid := strings.TrimSpace(string(b))
	if uid == "4294967295" || uid == "" {
		return "?"
	}
	name := userFromUID(uid)
	if name == "" {
		return uid
	}
	return name
}

var uidCache = map[string]string{}

func userFromUID(uid string) string {
	if n, ok := uidCache[uid]; ok {
		return n
	}
	b, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) > 2 && parts[2] == uid {
			uidCache[uid] = parts[0]
			return parts[0]
		}
	}
	uidCache[uid] = ""
	return ""
}

// Mib returns a human-readable MiB string.
func Mib(kib uint64) string {
	return fmt.Sprintf("%.1f", float64(kib)/1024)
}
