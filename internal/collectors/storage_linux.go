package collectors

import (
	"context"
	"fmt"
	"syscall"
	"time"
)

// StorageCollector stats the root filesystem via statfs. No exec, no parsing.
type StorageCollector struct{ Path string }

func (c *StorageCollector) Name() string { return "storage" }

func (c *StorageCollector) Collect(_ context.Context) (Snapshot, error) {
	path := c.Path
	if path == "" {
		path = "/"
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Snapshot{}, fmt.Errorf("collect storage metrics for %s: %w", path, err)
	}
	m := storageFromStatfs(path, st.Blocks, st.Bfree, st.Bavail, uint64(st.Bsize))
	return Snapshot{At: time.Now(), Available: true, Summary: formatStorageSummary(m)}, nil
}

func storageFromStatfs(path string, blocks, bfree, bavail, bsize uint64) StorageMetrics {
	total := blocks * bsize
	free := bavail * bsize
	used := (blocks - min(bfree, blocks)) * bsize
	var usage float64
	if total > 0 {
		usage = 100 * float64(used) / float64(total)
	}
	return StorageMetrics{Path: path, Total: total, Used: used, Free: free, Usage: usage}
}

func formatStorageSummary(m StorageMetrics) string {
	return fmt.Sprintf("%s %.1f/%.1f GiB (%.0f%%)",
		m.Path, float64(m.Used)/(1<<30), float64(m.Total)/(1<<30), m.Usage)
}
