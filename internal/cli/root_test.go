package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hieulvt/osinfo/internal/collector"
)

type stubCollector struct {
	osInfo  *collector.OSInfo
	cpuInfo *collector.CPUInfo
	memInfo *collector.MemoryInfo
	disks   []collector.DiskInfo
	err     error
}

func (s *stubCollector) OS(context.Context) (*collector.OSInfo, error) {
	return s.osInfo, s.err
}

func (s *stubCollector) CPU(context.Context) (*collector.CPUInfo, error) {
	return s.cpuInfo, s.err
}

func (s *stubCollector) Memory(context.Context) (*collector.MemoryInfo, error) {
	return s.memInfo, s.err
}

func (s *stubCollector) Disks(_ context.Context, path string) ([]collector.DiskInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	if path != "" {
		return []collector.DiskInfo{{Device: path, Mountpoint: path, Filesystem: "ext4", TotalBytes: 1024}}, nil
	}
	return s.disks, nil
}

func (s *stubCollector) All(context.Context, string) (*collector.SystemReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &collector.SystemReport{
		Timestamp: time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
		OS:        s.osInfo,
		CPU:       s.cpuInfo,
		Memory:    s.memInfo,
		Disks:     s.disks,
	}, nil
}

func newStub() *stubCollector {
	return &stubCollector{
		osInfo: &collector.OSInfo{
			Hostname:        "test-host",
			OS:              "linux",
			Platform:        "ubuntu",
			PlatformFamily:  "debian",
			PlatformVersion: "24.04",
			KernelVersion:   "6.8.0",
			KernelArch:      "x86_64",
			UptimeSeconds:   120,
			UptimeHuman:     "2m 0s",
		},
		cpuInfo: &collector.CPUInfo{
			ModelName:     "Test CPU",
			PhysicalCores: 4,
			LogicalCores:  8,
			Mhz:           3000,
			UsagePercent:  15.0,
		},
		memInfo: &collector.MemoryInfo{
			TotalBytes:     8 * 1024 * 1024 * 1024,
			AvailableBytes: 4 * 1024 * 1024 * 1024,
			UsedBytes:      4 * 1024 * 1024 * 1024,
			UsedPercent:    50.0,
		},
		disks: []collector.DiskInfo{
			{Device: "/dev/vda1", Mountpoint: "/", Filesystem: "ext4", TotalBytes: 50 * 1024 * 1024 * 1024},
		},
	}
}

func execCLI(t *testing.T, col collector.Collector, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := NewRootCmd(col, &stdout, &stderr)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	return stdout.String(), err
}

func TestCLI_Subcommands(t *testing.T) {
	stub := newStub()

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, "OS INFORMATION"},
		{"all", []string{"all"}, "DISK VOLUMES"},
		{"os", []string{"os"}, "test-host"},
		{"cpu", []string{"cpu"}, "Test CPU"},
		{"ram", []string{"ram"}, "8.00 GiB"},
		{"mem alias", []string{"mem"}, "MEMORY INFORMATION"},
		{"disk", []string{"disk"}, "/dev/vda1"},
		{"disk path", []string{"disk", "--path", "/var"}, "/var"},
		{"version", []string{"version"}, "osinfo "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := execCLI(t, stub, tc.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.want, out)
			}
		})
	}
}

func TestCLI_JSONOutput(t *testing.T) {
	stub := newStub()

	out, err := execCLI(t, stub, "os", "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var report collector.SystemReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if report.OS == nil || report.OS.Hostname != "test-host" {
		t.Fatalf("unexpected report: %+v", report)
	}

	verOut, err := execCLI(t, stub, "version", "-o", "json")
	if err != nil {
		t.Fatalf("unexpected version json error: %v", err)
	}
	var ver map[string]any
	if err := json.Unmarshal([]byte(verOut), &ver); err != nil {
		t.Fatalf("invalid version json: %v", err)
	}
	if _, ok := ver["version"]; !ok {
		t.Fatalf("expected version key in json output: %v", ver)
	}
}

func TestCLI_ErrorCases(t *testing.T) {
	stub := newStub()

	if _, err := execCLI(t, stub, "os", "-o", "xml"); err == nil {
		t.Fatal("expected error for invalid output format")
	}
	if _, err := execCLI(t, stub, "os", "--timeout", "0s"); err == nil {
		t.Fatal("expected error for non-positive timeout")
	}

	failing := &stubCollector{err: errors.New("boom")}
	if _, err := execCLI(t, failing, "cpu"); err == nil {
		t.Fatal("expected error when collector fails")
	}
}

