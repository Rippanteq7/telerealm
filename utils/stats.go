package utils

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

var startTime = time.Now()

type ServerStats struct {
	Uptime    string `json:"uptime"`
	Hostname  string `json:"hostname"`
	PID       int    `json:"pid"`
	PPID      int    `json:"ppid"`
	WorkDir   string `json:"work_dir"`
	StartedAt string `json:"started_at"`

	GoVersion    string `json:"go_version"`
	Compiler     string `json:"compiler"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCgoCall   int64  `json:"num_cgo_call"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	GOROOT       string `json:"goroot"`

	MemoryUsage map[string]string `json:"memory_usage"`

	HeapStats map[string]string `json:"heap_stats"`

	GCStats map[string]any `json:"gc_stats"`

	BuildInfo map[string]any `json:"build_info,omitempty"`
}

func GetServerStats() ServerStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hostname, _ := os.Hostname()
	workDir, _ := os.Getwd()

	stats := ServerStats{
		Uptime:    FormatDuration(time.Since(startTime)),
		Hostname:  hostname,
		PID:       os.Getpid(),
		PPID:      os.Getppid(),
		WorkDir:   workDir,
		StartedAt: startTime.Format(time.RFC3339),

		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCgoCall:   runtime.NumCgoCall(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		GOROOT:       runtime.GOROOT(),

		MemoryUsage: map[string]string{
			"alloc":       FormatBytes(m.Alloc),
			"total_alloc": FormatBytes(m.TotalAlloc),
			"sys":         FormatBytes(m.Sys),
			"mallocs":     fmt.Sprintf("%d", m.Mallocs),
			"frees":       fmt.Sprintf("%d", m.Frees),
			"live_objs":   fmt.Sprintf("%d", m.Mallocs-m.Frees),
		},

		HeapStats: map[string]string{
			"heap_alloc":    FormatBytes(m.HeapAlloc),
			"heap_sys":      FormatBytes(m.HeapSys),
			"heap_idle":     FormatBytes(m.HeapIdle),
			"heap_inuse":    FormatBytes(m.HeapInuse),
			"heap_released": FormatBytes(m.HeapReleased),
			"heap_objects":  fmt.Sprintf("%d", m.HeapObjects),
			"stack_sys":     FormatBytes(m.StackSys),
			"stack_inuse":   FormatBytes(m.StackInuse),
			"m_span_inuse":  FormatBytes(m.MSpanInuse),
			"m_cache_inuse": FormatBytes(m.MCacheInuse),
		},

		GCStats: getGCStats(&m),

		BuildInfo: getBuildInfo(),
	}

	return stats
}

func getGCStats(m *runtime.MemStats) map[string]any {
	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)

	lastPause := "N/A"
	if len(gcStats.Pause) > 0 {
		lastPause = gcStats.Pause[0].String()
	}

	lastGC := "N/A"
	if m.LastGC > 0 {
		lastGC = time.Unix(0, int64(m.LastGC)).Format(time.RFC3339)
	}

	return map[string]any{
		"num_gc":          m.NumGC,
		"num_forced_gc":   m.NumForcedGC,
		"last_gc":         lastGC,
		"last_pause":      lastPause,
		"pause_total_ns":  FormatDuration(time.Duration(m.PauseTotalNs)),
		"gc_cpu_fraction": fmt.Sprintf("%.6f%%", m.GCCPUFraction*100),
		"next_gc":         FormatBytes(m.NextGC),
		"total_pauses":    len(gcStats.Pause),
		"enable_gc":       m.EnableGC,
		"debug_gc":        m.DebugGC,
	}
}

func getBuildInfo() map[string]any {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}

	deps := []map[string]string{}
	for _, dep := range info.Deps {
		deps = append(deps, map[string]string{
			"path":    dep.Path,
			"version": dep.Version,
		})
	}

	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}

	return map[string]any{
		"go_version": info.GoVersion,
		"path":       info.Path,
		"main": map[string]string{
			"path":    info.Main.Path,
			"version": info.Main.Version,
		},
		"deps":     deps,
		"settings": settings,
	}
}

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
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return d.String()
	}

	totalSeconds := int(d.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}
