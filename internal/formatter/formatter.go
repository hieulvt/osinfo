package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/hieulvt/osinfo/internal/collector"
)

// Format identifies a supported output rendering mode.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

// Formatter serializes a SystemReport into a target writer.
type Formatter interface {
	Format(w io.Writer, report *collector.SystemReport) error
}

// New returns a Formatter for the requested format name.
func New(format string) (Formatter, error) {
	switch Format(strings.ToLower(strings.TrimSpace(format))) {
	case "", FormatTable:
		return &TableFormatter{}, nil
	case FormatJSON:
		return &JSONFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported output format %q (expected table or json)", format)
	}
}

// JSONFormatter renders reports as pretty-printed JSON.
type JSONFormatter struct{}

// Format writes the report as indented JSON.
func (f *JSONFormatter) Format(w io.Writer, report *collector.SystemReport) error {
	if report == nil {
		return fmt.Errorf("cannot format nil report")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// TableFormatter renders reports as aligned human-readable tables.
type TableFormatter struct{}

// Format writes non-nil sections of the report as readable text tables.
func (f *TableFormatter) Format(w io.Writer, report *collector.SystemReport) error {
	if report == nil {
		return fmt.Errorf("cannot format nil report")
	}

	sectionPrinted := false
	addSpacer := func() {
		if sectionPrinted {
			fmt.Fprintln(w)
		}
		sectionPrinted = true
	}

	if report.OS != nil {
		addSpacer()
		renderOS(w, report.OS)
	}
	if report.CPU != nil {
		addSpacer()
		renderCPU(w, report.CPU)
	}
	if report.Memory != nil {
		addSpacer()
		renderMemory(w, report.Memory)
	}
	if len(report.Disks) > 0 {
		addSpacer()
		renderDisks(w, report.Disks)
	}

	return nil
}

func renderOS(w io.Writer, osInfo *collector.OSInfo) {
	fmt.Fprintln(w, "OS INFORMATION")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Hostname:\t%s\n", osInfo.Hostname)
	fmt.Fprintf(tw, "OS:\t%s\n", osInfo.OS)
	fmt.Fprintf(tw, "Platform:\t%s %s (%s)\n", osInfo.Platform, osInfo.PlatformVersion, osInfo.PlatformFamily)
	fmt.Fprintf(tw, "Kernel:\t%s (%s)\n", osInfo.KernelVersion, osInfo.KernelArch)
	fmt.Fprintf(tw, "Uptime:\t%s\n", osInfo.UptimeHuman)
	_ = tw.Flush()
}

func renderCPU(w io.Writer, cpuInfo *collector.CPUInfo) {
	fmt.Fprintln(w, "CPU INFORMATION")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Model:\t%s\n", cpuInfo.ModelName)
	fmt.Fprintf(tw, "Cores:\t%d physical / %d logical\n", cpuInfo.PhysicalCores, cpuInfo.LogicalCores)
	fmt.Fprintf(tw, "Clock:\t%.0f MHz\n", cpuInfo.Mhz)
	fmt.Fprintf(tw, "Usage:\t%.1f%%\n", cpuInfo.UsagePercent)
	_ = tw.Flush()
}

func renderMemory(w io.Writer, memInfo *collector.MemoryInfo) {
	fmt.Fprintln(w, "MEMORY INFORMATION")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "RAM Total:\t%s\n", FormatBytes(memInfo.TotalBytes))
	fmt.Fprintf(tw, "RAM Used:\t%s (%.1f%%)\n", FormatBytes(memInfo.UsedBytes), memInfo.UsedPercent)
	fmt.Fprintf(tw, "RAM Available:\t%s\n", FormatBytes(memInfo.AvailableBytes))
	fmt.Fprintf(tw, "Swap Total:\t%s\n", FormatBytes(memInfo.SwapTotalBytes))
	fmt.Fprintf(tw, "Swap Used:\t%s (%.1f%%)\n", FormatBytes(memInfo.SwapUsedBytes), memInfo.SwapUsedPercent)
	_ = tw.Flush()
}

func renderDisks(w io.Writer, disks []collector.DiskInfo) {
	fmt.Fprintln(w, "DISK VOLUMES")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "DEVICE\tMOUNTPOINT\tFS\tTOTAL\tUSED\tFREE\tUSE%")
	for _, d := range disks {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%.1f%%\n",
			d.Device,
			d.Mountpoint,
			d.Filesystem,
			FormatBytes(d.TotalBytes),
			FormatBytes(d.UsedBytes),
			FormatBytes(d.FreeBytes),
			d.UsedPercent,
		)
	}
	_ = tw.Flush()
}

// FormatBytes formats a byte count into human-readable IEC units.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.2f %s", float64(b)/float64(div), units[exp])
}

