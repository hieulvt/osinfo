package collector

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func newMockCollector() *SystemCollector {
	fixedTime := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	return &SystemCollector{
		p: providers{
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Hostname:        "prod-node-01",
					OS:              "linux",
					Platform:        "ubuntu",
					PlatformFamily:  "debian",
					PlatformVersion: "24.04",
					KernelVersion:   "6.8.0-45-generic",
					KernelArch:      "x86_64",
					Uptime:          90065, // 1d 1h 1m 5s
				}, nil
			},
			cpuInfo: func(context.Context) ([]cpu.InfoStat, error) {
				return []cpu.InfoStat{
					{ModelName: "  AMD EPYC 7763 64-Core Processor  ", Mhz: 2445.0},
				}, nil
			},
			cpuCounts: func(_ context.Context, logical bool) (int, error) {
				if logical {
					return 16, nil
				}
				return 8, nil
			},
			cpuPercent: func(context.Context, time.Duration, bool) ([]float64, error) {
				return []float64{23.4}, nil
			},
			virtualMem: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{
					Total:       32 * 1024 * 1024 * 1024,
					Available:   20 * 1024 * 1024 * 1024,
					Used:        12 * 1024 * 1024 * 1024,
					UsedPercent: 37.5,
				}, nil
			},
			swapMem: func(context.Context) (*mem.SwapMemoryStat, error) {
				return &mem.SwapMemoryStat{
					Total:       4 * 1024 * 1024 * 1024,
					Used:        512 * 1024 * 1024,
					UsedPercent: 12.5,
				}, nil
			},
			partitions: func(context.Context, bool) ([]disk.PartitionStat, error) {
				return []disk.PartitionStat{
					{Device: "/dev/nvme0n1p1", Mountpoint: "/", Fstype: "ext4"},
					{Device: "/dev/loop0", Mountpoint: "/snap/core22/1", Fstype: "squashfs"},
					{Device: "tmpfs", Mountpoint: "/run", Fstype: "tmpfs"},
					{Device: "/dev/nvme0n1p1", Mountpoint: "/", Fstype: "ext4"}, // duplicate
					{Device: "/dev/nvme1n1", Mountpoint: "/data", Fstype: "xfs"},
				}, nil
			},
			diskUsage: func(_ context.Context, path string) (*disk.UsageStat, error) {
				switch path {
				case "/":
					return &disk.UsageStat{
						Path:        "/",
						Fstype:      "ext4",
						Total:       100 * 1024 * 1024 * 1024,
						Used:        40 * 1024 * 1024 * 1024,
						Free:        60 * 1024 * 1024 * 1024,
						UsedPercent: 40.0,
					}, nil
				case "/data":
					return &disk.UsageStat{
						Path:        "/data",
						Fstype:      "xfs",
						Total:       500 * 1024 * 1024 * 1024,
						Used:        100 * 1024 * 1024 * 1024,
						Free:        400 * 1024 * 1024 * 1024,
						UsedPercent: 20.0,
					}, nil
				default:
					return nil, errors.New("not found")
				}
			},
			now:          func() time.Time { return fixedTime },
			cpuSampleDur: 0,
			netInterfaces: func() ([]net.Interface, error) {
				return []net.Interface{
					{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
					{Name: "eth0", Flags: net.FlagUp},
					{Name: "eth1", Flags: 0}, // down
				}, nil
			},
			ifaceAddrs: func(iface net.Interface) ([]net.Addr, error) {
				switch iface.Name {
				case "eth0":
					return []net.Addr{
						&net.IPNet{IP: net.ParseIP("192.168.1.50"), Mask: net.CIDRMask(24, 32)},
						&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(64, 128)}, // link-local skipped
						&net.IPNet{IP: net.ParseIP("2001:db8::10"), Mask: net.CIDRMask(64, 128)},
					}, nil
				default:
					return nil, nil
				}
			},
		},
	}
}

func TestSystemCollector_OS(t *testing.T) {
	col := newMockCollector()
	got, err := col.OS(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hostname != "prod-node-01" || got.Platform != "ubuntu" || got.UptimeHuman != "1d 1h 1m" {
		t.Fatalf("unexpected OSInfo: %+v", got)
	}
}

func TestSystemCollector_CPU(t *testing.T) {
	col := newMockCollector()
	got, err := col.CPU(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ModelName != "AMD EPYC 7763 64-Core Processor" {
		t.Fatalf("expected trimmed model name, got %q", got.ModelName)
	}
	if got.PhysicalCores != 8 || got.LogicalCores != 16 || got.UsagePercent != 23.4 {
		t.Fatalf("unexpected CPUInfo: %+v", got)
	}
}

func TestSystemCollector_Memory(t *testing.T) {
	col := newMockCollector()
	got, err := col.Memory(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalBytes != 32*1024*1024*1024 || got.SwapTotalBytes != 4*1024*1024*1024 {
		t.Fatalf("unexpected MemoryInfo: %+v", got)
	}
}

func TestSystemCollector_Disks(t *testing.T) {
	col := newMockCollector()

	disks, err := col.Disks(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disks) != 2 {
		t.Fatalf("expected 2 filtered/deduplicated disks, got %d: %+v", len(disks), disks)
	}
	if disks[0].Mountpoint != "/" || disks[1].Mountpoint != "/data" {
		t.Fatalf("unexpected mountpoints: %+v", disks)
	}

	single, err := col.Disks(context.Background(), "/data")
	if err != nil {
		t.Fatalf("unexpected error for explicit path: %v", err)
	}
	if len(single) != 1 || single[0].Mountpoint != "/data" {
		t.Fatalf("unexpected single disk result: %+v", single)
	}
}

func TestSystemCollector_Network(t *testing.T) {
	col := newMockCollector()
	addrs, err := col.Network(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 2 {
		t.Fatalf("expected 2 addresses (IPv4 + global IPv6 on eth0), got %d: %+v", len(addrs), addrs)
	}
	if addrs[0].Interface != "eth0" || addrs[0].IP != "192.168.1.50" || addrs[0].Version != "IPv4" {
		t.Fatalf("unexpected first address: %+v", addrs[0])
	}
	if addrs[1].IP != "2001:db8::10" || addrs[1].Version != "IPv6" {
		t.Fatalf("unexpected second address: %+v", addrs[1])
	}
}

func TestSystemCollector_All(t *testing.T) {
	col := newMockCollector()
	report, err := col.All(context.Background(), "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.OS == nil || report.CPU == nil || report.Memory == nil || len(report.Disks) != 2 || len(report.Network) != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}

	reportWithNet, err := col.All(context.Background(), "", true)
	if err != nil {
		t.Fatalf("unexpected error with network: %v", err)
	}
	if len(reportWithNet.Network) != 2 {
		t.Fatalf("expected 2 network addresses when includeNetwork=true, got %+v", reportWithNet.Network)
	}

	// Verify error aggregation when one provider fails.
	col.p.cpuPercent = func(context.Context, time.Duration, bool) ([]float64, error) {
		return nil, errors.New("cpu read failed")
	}
	report, err = col.All(context.Background(), "", false)
	if err == nil {
		t.Fatal("expected error when cpu provider fails")
	}
	if report.OS == nil || report.Memory == nil || report.CPU != nil {
		t.Fatalf("expected partial report with nil CPU, got %+v", report)
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		seconds uint64
		want    string
	}{
		{45, "0m 45s"},
		{3665, "1h 1m"},
		{90065, "1d 1h 1m"},
	}
	for _, tc := range cases {
		if got := formatUptime(tc.seconds); got != tc.want {
			t.Errorf("formatUptime(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

