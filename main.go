// attendance-collector is a small local application that pulls employees and
// attendance punches from ZKTeco device(s) and stores them (deduplicated) in a
// database. It has no web UI — just a CLI with one-shot commands and a `run`
// daemon that polls on a staggered schedule.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"attendance-collector/internal/collector"
	"attendance-collector/internal/config"
	"attendance-collector/internal/db"
	"attendance-collector/internal/zk"

	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}

	gdb, err := db.Open(cfg)
	if err != nil {
		fatalf("db: %v", err)
	}

	col := collector.New(gdb, cfg)

	switch cmd {
	case "pull":
		runOnce(cfg, args, func(d config.DeviceConfig) error { return col.PullAttendance(d) })
		if err := col.BackfillLogDirections(); err != nil {
			fmt.Fprintf(os.Stderr, "direction backfill: %v\n", err)
		}
	case "sync-employees":
		runOnce(cfg, args, func(d config.DeviceConfig) error { return col.SyncEmployees(d) })
	case "test":
		testConnections(cfg)
	case "clear-device":
		clearDevices(cfg, col, args)
	case "run":
		runDaemon(cfg, col, gdb)
	default:
		usage()
		os.Exit(1)
	}
}

// runOnce runs fn against the selected devices, once each.
func runOnce(cfg *config.Config, args []string, fn func(config.DeviceConfig) error) {
	filter := deviceFilter(args)
	devices := selectDevices(cfg, filter)
	if len(devices) == 0 {
		fatalf("no device matched %q", filter)
	}
	for _, d := range devices {
		if err := fn(d); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] error: %v\n", d.Name, err)
		}
	}
}

func testConnections(cfg *config.Config) {
	for _, d := range cfg.Devices {
		fmt.Printf("Testing %s (%s) ... ", d.Name, d.Address())
		t, err := zkTime(cfg, d)
		if err != nil {
			fmt.Printf("FAILED — %v\n", err)
			continue
		}
		fmt.Printf("OK — device time: %s\n", t.Format("2006-01-02 15:04:05"))
	}
}

func clearDevices(cfg *config.Config, col *collector.Collector, args []string) {
	fatalf("clear-device command is permanently disabled: hardware device data deletion is prevented for safety.")
}

// runDaemon polls every device on its own interval, staggered so two devices
// never talk to the network at the same time. Device 1 runs at offset +0,
// Device 2 runs at offset +5m (or STAGGER_MINUTES), and both repeat every
// POLL_INTERVAL_MINUTES (default 10m).
func runDaemon(cfg *config.Config, col *collector.Collector, _ *gorm.DB) {
	fmt.Println("======================================================================")
	fmt.Printf("Attendance Collector Daemon Running (Production Mode)\n")
	fmt.Printf("  • Devices Configured: %d\n", len(cfg.Devices))
	fmt.Printf("  • Poll Interval:      %s (every %d minutes)\n", cfg.PollInterval, int(cfg.PollInterval.Minutes()))
	fmt.Printf("  • Device Stagger:     %s (device 2 pulls %d minutes after device 1)\n", cfg.Stagger, int(cfg.Stagger.Minutes()))
	fmt.Println("  • Schedule Timeline:")
	for i, d := range cfg.Devices {
		offset := time.Duration(i) * cfg.Stagger
		fmt.Printf("      - Device #%d [%s] (%s): Initial start after +%s, then every %s\n",
			i+1, d.Name, d.Address(), offset, cfg.PollInterval)
	}
	fmt.Println("======================================================================")

	if err := col.BackfillLogDirections(); err != nil {
		fmt.Fprintf(os.Stderr, "direction backfill: %v\n", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Attendance pollers, one per device, offset by stagger.
	for i, d := range cfg.Devices {
		wg.Add(1)
		go func(idx int, dev config.DeviceConfig) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "[%s] CRITICAL PANIC RECOVERED: %v\n", dev.Name, r)
				}
			}()

			initialOffset := time.Duration(idx) * cfg.Stagger
			if initialOffset > 0 {
				fmt.Printf("[%s] schedule initialized: waiting %s stagger offset before first pull...\n", dev.Name, initialOffset)
			}

			select {
			case <-time.After(initialOffset):
			case <-ctx.Done():
				return
			}

			pull := func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Fprintf(os.Stderr, "[%s] PANIC during pull: %v\n", dev.Name, r)
					}
				}()
				if err := col.PullAttendance(dev); err != nil {
					fmt.Fprintf(os.Stderr, "[%s] pull error: %v\n", dev.Name, err)
				}
			}

			fmt.Printf("[%s] starting first attendance pull\n", dev.Name)
			pull()

			t := time.NewTicker(cfg.PollInterval)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					fmt.Printf("[%s] scheduled ticker triggered (interval: %s)\n", dev.Name, cfg.PollInterval)
					pull()
				case <-ctx.Done():
					fmt.Printf("[%s] poller stopping cleanly\n", dev.Name)
					return
				}
			}
		}(i, d)
	}

	// Employee sync at start + every 6 hours.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[employee-sync] CRITICAL PANIC RECOVERED: %v\n", r)
			}
		}()

		sync := func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "[employee-sync] PANIC during sync: %v\n", r)
				}
			}()
			for _, d := range cfg.Devices {
				if err := col.SyncEmployees(d); err != nil {
					fmt.Fprintf(os.Stderr, "[%s] sync error: %v\n", d.Name, err)
				}
			}
		}

		sync()
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				sync()
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	fmt.Println("\nShutdown signal received. Waiting for active operations to complete...")
	wg.Wait()
	fmt.Println("Attendance collector daemon shut down cleanly.")
}

// --- helpers ---

func zkTime(cfg *config.Config, d config.DeviceConfig) (time.Time, error) {
	return zk.New(d, cfg).Time()
}

func deviceFilter(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--device" || a == "-device" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		if strings.HasPrefix(a, "--device=") {
			return strings.TrimPrefix(a, "--device=")
		}
	}
	return ""
}

// selectDevices filters by direction ("in"/"out"), IP, or name substring.
// "in"/"out" match the direction exactly (so "in" doesn't match "Inside").
func selectDevices(cfg *config.Config, filter string) []config.DeviceConfig {
	if filter == "" {
		return cfg.Devices
	}
	f := strings.ToLower(filter)
	var out []config.DeviceConfig
	for _, d := range cfg.Devices {
		var match bool
		switch f {
		case "in", "out":
			match = strings.EqualFold(d.Direction, f)
		default:
			match = d.IP == filter || strings.Contains(strings.ToLower(d.Name), f)
		}
		if match {
			out = append(out, d)
		}
	}
	return out
}

func yes() bool {
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`attendance-collector — pull ZKTeco employees & attendance into a database

Usage:
  attendance-collector <command> [flags]

Commands:
  pull            [--device=<in|out|ip|name>]   Pull attendance punches (dedup) and store
  sync-employees  [--device=...]                Sync enrolled users into employees table
  test                                          Test connectivity (prints device time)
  run                                           Daemon: poll on a staggered schedule

Configuration is read from environment variables / a .env file (see .env.example).
`)
}
