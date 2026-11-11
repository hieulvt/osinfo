package collector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

const defaultCPUSampleInterval = 200 * time.Millisecond

// ignoredFSTypes filters out virtual/pseudo filesystems that clutter physical disk reports.
var ignoredFSTypes = map[string]struct{}{
	"squashfs": {},
	"tmpfs":    {},
	"devtmpfs": {},
	"overlay":  {},
	"proc":     {},
	"sysfs":    {},
	"cgroup":   {},
	"cgroup2":  {},
	"autofs":   {},
	"efivarfs": {},
}

type providers struct {
	hostInfo     func(context.Context) (*host.InfoStat, error)
	cpuInfo      func(context.Context) ([]cpu.InfoStat, error)
	cpuCounts    func(context.Context, bool) (int, error)
	cpuPercent   func(context.Context, time.Duration, bool) ([]float64, error)
	virtualMem   func(context.Context) (*mem.VirtualMemoryStat, error)
	swapMem      func(context.Context) (*mem.SwapMemoryStat, error)
	partitions   func(context.Context, bool) ([]disk.PartitionStat, error)
	diskUsage    func(context.Context, string) (*disk.UsageStat, error)
	now          func() time.Time
	cpuSampleDur time.Duration
}

// SystemCollector gathers live system telemetry from the host OS via gopsutil.
type SystemCollector struct {
	p providers
}

// NewSystemCollector returns a SystemCollector configured with live OS providers.
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{
		p: providers{
			hostInfo:     host.InfoWithContext,
			cpuInfo:      cpu.InfoWithContext,
			cpuCounts:    cpu.CountsWithContext,
			cpuPercent:   cpu.PercentWithContext,
			virtualMem:   mem.VirtualMemoryWithContext,
			swapMem:      mem.SwapMemoryWithContext,
			partitions:   disk.PartitionsWithContext,
			diskUsage:    disk.UsageWithContext,
			now:          time.Now,
			cpuSampleDur: defaultCPUSampleInterval,
		},
	}
}

// OS collects operating system, kernel, hostname, and uptime details.
func (c *SystemCollector) OS(ctx context.Context) (*OSInfo, error) {
	info, err := c.p.hostInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect os info: %w", err)
	}

	return &OSInfo{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformFamily:  info.PlatformFamily,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		KernelArch:      info.KernelArch,
		UptimeSeconds:   info.Uptime,
		UptimeHuman:     formatUptime(info.Uptime),
	}, nil
}

// CPU collects processor model, core counts, frequency, and sampled utilization.
func (c *SystemCollector) CPU(ctx context.Context) (*CPUInfo, error) {
	infos, err := c.p.cpuInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect cpu info: %w", err)
	}

	physical, err := c.p.cpuCounts(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("collect physical cpu count: %w", err)
	}

	logical, err := c.p.cpuCounts(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("collect logical cpu count: %w", err)
	}

	pcts, err := c.p.cpuPercent(ctx, c.p.cpuSampleDur, false)
	if err != nil {
		return nil, fmt.Errorf("collect cpu usage: %w", err)
	}

	out := &CPUInfo{
		PhysicalCores: physical,
		LogicalCores:  logical,
	}
	if len(infos) > 0 {
		out.ModelName = strings.TrimSpace(infos[0].ModelName)
		out.Mhz = infos[0].Mhz
	}
	if len(pcts) > 0 {
		out.UsagePercent = pcts[0]
	}

	return out, nil
}

// Memory collects physical RAM and swap memory statistics.
func (c *SystemCollector) Memory(ctx context.Context) (*MemoryInfo, error) {
	vm, err := c.p.virtualMem(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect virtual memory: %w", err)
	}

	swap, err := c.p.swapMem(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect swap memory: %w", err)
	}

	return &MemoryInfo{
		TotalBytes:      vm.Total,
		AvailableBytes:  vm.Available,
		UsedBytes:       vm.Used,
		UsedPercent:     vm.UsedPercent,
		SwapTotalBytes:  swap.Total,
		SwapUsedBytes:   swap.Used,
		SwapUsedPercent: swap.UsedPercent,
	}, nil
}

// Disks collects usage statistics for a specific path or all physical mounted volumes.
func (c *SystemCollector) Disks(ctx context.Context, path string) ([]DiskInfo, error) {
	if path != "" {
		usage, err := c.p.diskUsage(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("collect disk usage for %q: %w", path, err)
		}
		return []DiskInfo{
			{
				Device:      path,
				Mountpoint:  usage.Path,
				Filesystem:  usage.Fstype,
				TotalBytes:  usage.Total,
				UsedBytes:   usage.Used,
				FreeBytes:   usage.Free,
				UsedPercent: usage.UsedPercent,
			},
		}, nil
	}

	parts, err := c.p.partitions(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("list disk partitions: %w", err)
	}

	seen := make(map[string]struct{}, len(parts))
	disks := make([]DiskInfo, 0, len(parts))
	var errs []error

	for _, p := range parts {
		if shouldSkipPartition(p) {
			continue
		}
		if _, ok := seen[p.Mountpoint]; ok {
			continue
		}
		seen[p.Mountpoint] = struct{}{}

		usage, uErr := c.p.diskUsage(ctx, p.Mountpoint)
		if uErr != nil {
			errs = append(errs, fmt.Errorf("disk usage %q: %w", p.Mountpoint, uErr))
			continue
		}
		if usage.Total == 0 {
			continue
		}

		fs := p.Fstype
		if fs == "" {
			fs = usage.Fstype
		}

		disks = append(disks, DiskInfo{
			Device:      p.Device,
			Mountpoint:  p.Mountpoint,
			Filesystem:  fs,
			TotalBytes:  usage.Total,
			UsedBytes:   usage.Used,
			FreeBytes:   usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	if len(disks) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return disks, nil
}

// All collects OS, CPU, Memory, and Disk metrics concurrently into a SystemReport.
func (c *SystemCollector) All(ctx context.Context, diskPath string) (*SystemReport, error) {
	report := &SystemReport{
		Timestamp: c.p.now().UTC(),
	}

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		err error
	)

	recordErr := func(e error) {
		if e == nil {
			return
		}
		mu.Lock()
		err = errors.Join(err, e)
		mu.Unlock()
	}

	wg.Add(4)
	go func() {
		defer wg.Done()
		osInfo, e := c.OS(ctx)
		if e != nil {
			recordErr(e)
			return
		}
		report.OS = osInfo
	}()

	go func() {
		defer wg.Done()
		cpuInfo, e := c.CPU(ctx)
		if e != nil {
			recordErr(e)
			return
		}
		report.CPU = cpuInfo
	}()

	go func() {
		defer wg.Done()
		memInfo, e := c.Memory(ctx)
		if e != nil {
			recordErr(e)
			return
		}
		report.Memory = memInfo
	}()

	go func() {
		defer wg.Done()
		disks, e := c.Disks(ctx, diskPath)
		if e != nil {
			recordErr(e)
			return
		}
		report.Disks = disks
	}()

	wg.Wait()

	if err != nil {
		return report, err
	}
	return report, nil
}

func shouldSkipPartition(p disk.PartitionStat) bool {
	if p.Mountpoint == "" {
		return true
	}
	if _, skip := ignoredFSTypes[strings.ToLower(p.Fstype)]; skip {
		return true
	}
	if strings.HasPrefix(p.Mountpoint, "/snap/") || strings.HasPrefix(p.Device, "/dev/loop") {
		return true
	}
	return false
}

func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm %ds", minutes, seconds%60)
	}
}

