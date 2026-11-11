package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hieulvt/osinfo/internal/collector"
)

func sampleReport() *collector.SystemReport {
	return &collector.SystemReport{
		Timestamp: time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
		OS: &collector.OSInfo{
			Hostname:        "web-01",
			OS:              "linux",
			Platform:        "ubuntu",
			PlatformFamily:  "debian",
			PlatformVersion: "24.04",
			KernelVersion:   "6.8.0-45-generic",
			KernelArch:      "x86_64",
			UptimeSeconds:   3660,
			UptimeHuman:     "1h 1m",
		},
		CPU: &collector.CPUInfo{
			ModelName:     "Intel Xeon",
			PhysicalCores: 4,
			LogicalCores:  8,
			Mhz:           2600,
			UsagePercent:  12.5,
		},
		Memory: &collector.MemoryInfo{
			TotalBytes:      16 * 1024 * 1024 * 1024,
			AvailableBytes:  10 * 1024 * 1024 * 1024,
			UsedBytes:       6 * 1024 * 1024 * 1024,
			UsedPercent:     37.5,
			SwapTotalBytes:  2 * 1024 * 1024 * 1024,
			SwapUsedBytes:   0,
			SwapUsedPercent: 0,
		},
		Disks: []collector.DiskInfo{
			{
				Device:      "/dev/sda1",
				Mountpoint:  "/",
				Filesystem:  "ext4",
				TotalBytes:  100 * 1024 * 1024 * 1024,
				UsedBytes:   25 * 1024 * 1024 * 1024,
				FreeBytes:   75 * 1024 * 1024 * 1024,
				UsedPercent: 25.0,
			},
		},
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		bytes uint64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KiB"},
		{1536 * 1024, "1.50 MiB"},
		{16 * 1024 * 1024 * 1024, "16.00 GiB"},
	}
	for _, tc := range cases {
		if got := FormatBytes(tc.bytes); got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestTableFormatter(t *testing.T) {
	fmtr, err := New("table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := fmtr.Format(&buf, sampleReport()); err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"OS INFORMATION",
		"web-01",
		"ubuntu 24.04",
		"CPU INFORMATION",
		"Intel Xeon",
		"MEMORY INFORMATION",
		"16.00 GiB",
		"DISK VOLUMES",
		"/dev/sda1",
		"100.00 GiB",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestJSONFormatter(t *testing.T) {
	fmtr, err := New("json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := fmtr.Format(&buf, sampleReport()); err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	var decoded collector.SystemReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if decoded.OS == nil || decoded.OS.Hostname != "web-01" {
		t.Fatalf("unexpected decoded report: %+v", decoded)
	}
}

func TestNewFormatter_Errors(t *testing.T) {
	if _, err := New("yaml"); err == nil {
		t.Fatal("expected error for unsupported format")
	}
	fmtr := &TableFormatter{}
	if err := fmtr.Format(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("expected error for nil report")
	}
}

