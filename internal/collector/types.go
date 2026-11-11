package collector

import (
	"context"
	"time"
)

// SystemReport aggregates collected system metrics into a single snapshot.
type SystemReport struct {
	Timestamp time.Time   `json:"timestamp"`
	OS        *OSInfo     `json:"os,omitempty"`
	CPU       *CPUInfo    `json:"cpu,omitempty"`
	Memory    *MemoryInfo `json:"memory,omitempty"`
	Disks     []DiskInfo  `json:"disks,omitempty"`
}

// OSInfo represents operating system and host metadata.
type OSInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformFamily  string `json:"platform_family"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	KernelArch      string `json:"kernel_arch"`
	UptimeSeconds   uint64 `json:"uptime_seconds"`
	UptimeHuman     string `json:"uptime_human"`
}

// CPUInfo represents processor hardware details and utilization.
type CPUInfo struct {
	ModelName     string  `json:"model_name"`
	PhysicalCores int     `json:"physical_cores"`
	LogicalCores  int     `json:"logical_cores"`
	Mhz           float64 `json:"mhz"`
	UsagePercent  float64 `json:"usage_percent"`
}

// MemoryInfo represents physical RAM and swap usage in bytes.
type MemoryInfo struct {
	TotalBytes      uint64  `json:"total_bytes"`
	AvailableBytes  uint64  `json:"available_bytes"`
	UsedBytes       uint64  `json:"used_bytes"`
	UsedPercent     float64 `json:"used_percent"`
	SwapTotalBytes  uint64  `json:"swap_total_bytes"`
	SwapUsedBytes   uint64  `json:"swap_used_bytes"`
	SwapUsedPercent float64 `json:"swap_used_percent"`
}

// DiskInfo represents storage capacity and usage for a mounted volume.
type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Filesystem  string  `json:"filesystem"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// Collector defines the contract for gathering system telemetry.
type Collector interface {
	OS(ctx context.Context) (*OSInfo, error)
	CPU(ctx context.Context) (*CPUInfo, error)
	Memory(ctx context.Context) (*MemoryInfo, error)
	Disks(ctx context.Context, path string) ([]DiskInfo, error)
	All(ctx context.Context, diskPath string) (*SystemReport, error)
}

