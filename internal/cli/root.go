package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hieulvt/osinfo/internal/collector"
	"github.com/hieulvt/osinfo/internal/formatter"
	"github.com/hieulvt/osinfo/internal/version"
)

const defaultTimeout = 5 * time.Second

type options struct {
	output   string
	timeout  time.Duration
	diskPath string
}

// NewRootCmd builds the root osinfo CLI command with injected collector and output streams.
func NewRootCmd(col collector.Collector, stdout, stderr io.Writer) *cobra.Command {
	opts := &options{
		output:  string(formatter.FormatTable),
		timeout: defaultTimeout,
	}

	rootCmd := &cobra.Command{
		Use:           "osinfo",
		Short:         "Inspect host OS, CPU, memory, and disk volume information",
		Long:          "osinfo is a lightweight CLI tool for inspecting operating system, CPU, memory, and disk volume telemetry.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAll(cmd.Context(), col, opts, stdout)
		},
	}

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.PersistentFlags().StringVarP(&opts.output, "output", "o", opts.output, "Output format (table or json)")
	rootCmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", opts.timeout, "Maximum time allowed for metric collection")

	rootCmd.AddCommand(
		newAllCmd(col, opts, stdout),
		newOSCmd(col, opts, stdout),
		newCPUCmd(col, opts, stdout),
		newRAMCmd(col, opts, stdout),
		newDiskCmd(col, opts, stdout),
		newVersionCmd(opts, stdout),
	)

	return rootCmd
}

func newAllCmd(col collector.Collector, opts *options, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Display all system information (OS, CPU, RAM, Disk)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAll(cmd.Context(), col, opts, stdout)
		},
	}
	cmd.Flags().StringVar(&opts.diskPath, "path", "", "Optional path to inspect instead of all mounted disks")
	return cmd
}

func newOSCmd(col collector.Collector, opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "os",
		Short: "Display operating system version, kernel, and uptime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithReport(cmd.Context(), opts, stdout, func(ctx context.Context) (*collector.SystemReport, error) {
				info, err := col.OS(ctx)
				if err != nil {
					return nil, err
				}
				return &collector.SystemReport{
					Timestamp: time.Now().UTC(),
					OS:        info,
				}, nil
			})
		},
	}
}

func newCPUCmd(col collector.Collector, opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "cpu",
		Short: "Display CPU model, core counts, clock speed, and usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithReport(cmd.Context(), opts, stdout, func(ctx context.Context) (*collector.SystemReport, error) {
				info, err := col.CPU(ctx)
				if err != nil {
					return nil, err
				}
				return &collector.SystemReport{
					Timestamp: time.Now().UTC(),
					CPU:       info,
				}, nil
			})
		},
	}
}

func newRAMCmd(col collector.Collector, opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:     "ram",
		Aliases: []string{"mem", "memory"},
		Short:   "Display physical RAM and swap memory usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithReport(cmd.Context(), opts, stdout, func(ctx context.Context) (*collector.SystemReport, error) {
				info, err := col.Memory(ctx)
				if err != nil {
					return nil, err
				}
				return &collector.SystemReport{
					Timestamp: time.Now().UTC(),
					Memory:    info,
				}, nil
			})
		},
	}
}

func newDiskCmd(col collector.Collector, opts *options, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disk",
		Aliases: []string{"disks"},
		Short:   "Display mounted disk volume capacities and usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithReport(cmd.Context(), opts, stdout, func(ctx context.Context) (*collector.SystemReport, error) {
				disks, err := col.Disks(ctx, opts.diskPath)
				if err != nil {
					return nil, err
				}
				return &collector.SystemReport{
					Timestamp: time.Now().UTC(),
					Disks:     disks,
				}, nil
			})
		},
	}
	cmd.Flags().StringVar(&opts.diskPath, "path", "", "Inspect disk usage for a specific filesystem path")
	return cmd
}

func newVersionCmd(opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display build and runtime version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			info := version.Get()
			switch formatter.Format(strings.ToLower(strings.TrimSpace(opts.output))) {
			case formatter.FormatJSON:
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			case "", formatter.FormatTable:
				_, err := fmt.Fprintln(stdout, info.String())
				return err
			default:
				return fmt.Errorf("unsupported output format %q (expected table or json)", opts.output)
			}
		},
	}
}

func runAll(ctx context.Context, col collector.Collector, opts *options, stdout io.Writer) error {
	return runWithReport(ctx, opts, stdout, func(c context.Context) (*collector.SystemReport, error) {
		return col.All(c, opts.diskPath)
	})
}

func runWithReport(
	ctx context.Context,
	opts *options,
	stdout io.Writer,
	collectFn func(context.Context) (*collector.SystemReport, error),
) error {
	if opts.timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %s", opts.timeout)
	}

	fmtr, err := formatter.New(opts.output)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	report, err := collectFn(ctx)
	if err != nil {
		return err
	}

	return fmtr.Format(stdout, report)
}
