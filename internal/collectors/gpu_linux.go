package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// drmRoot is /sys/class/drm in production, overridable in tests.
var drmRoot = "/sys/class/drm"

// execSmi runs nvidia-smi. Overridable in tests.
var execSmi = func(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=name,utilization.gpu,temperature.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GPUCollector tries NVML (dlopen, no cgo), then nvidia-smi,
// then AMD sysfs. Unsupported hardware yields an unavailable
// Snapshot, never an error.
type GPUCollector struct {
	mu     sync.Mutex
	nvml   *nvmlLib
	probed bool
}

func (c *GPUCollector) Name() string { return "gpu" }

func (c *GPUCollector) Collect(ctx context.Context) (Snapshot, error) {
	if m, err := c.tryNVML(); err == nil {
		return Snapshot{At: time.Now(), Available: true, Summary: formatGPUSummary(m)}, nil
	}
	if out, err := execSmi(ctx); err == nil {
		if m, err := parseSmiCSV(out); err == nil {
			return Snapshot{At: time.Now(), Available: true, Summary: formatGPUSummary(m)}, nil
		}
	}
	if m, err := amdGPU(); err == nil {
		return Snapshot{At: time.Now(), Available: true, Summary: formatGPUSummary(m)}, nil
	}
	return Snapshot{At: time.Now(), Available: false, Summary: "GPU information unavailable"}, nil
}

func (c *GPUCollector) tryNVML() (GPUMetrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.probed {
		c.nvml, _ = nvmlOpen(nvmlLibPath)
		c.probed = true
	}
	if c.nvml == nil {
		return GPUMetrics{}, fmt.Errorf("nvml unavailable")
	}
	return c.nvml.firstGPU()
}

func formatGPUSummary(m GPUMetrics) string {
	var b strings.Builder
	b.WriteString(m.Model)
	fmt.Fprintf(&b, "\n%.0f%%", m.Usage)
	if m.Temperature != nil {
		fmt.Fprintf(&b, " · %.0f°C", *m.Temperature)
	}
	if m.VRAMTotal > 0 {
		fmt.Fprintf(&b, " · %.1f/%.1f GiB", float64(m.VRAMUsed)/(1<<30), float64(m.VRAMTotal)/(1<<30))
	}
	return b.String()
}

// --- NVML via dlopen (purego, no cgo/headers) ---

type nvmlUtil struct{ Gpu, Memory uint32 }
type nvmlMem struct{ Total, Free, Used uint64 }

type nvmlLib struct {
	init   func() int32
	count  func(*uint32) int32
	handle func(uint32, *unsafe.Pointer) int32
	name   func(unsafe.Pointer, *byte, uint32) int32
	util   func(unsafe.Pointer, *nvmlUtil) int32
	temp   func(unsafe.Pointer, int32, *uint32) int32
	mem    func(unsafe.Pointer, *nvmlMem) int32
}

// nvmlLibPath is overridable in tests to force the fallback path.
var nvmlLibPath = "libnvidia-ml.so.1"

func nvmlOpen(path string) (*nvmlLib, error) {
	lib, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, err
	}
	n := &nvmlLib{}
	syms := []struct {
		name string
		fn   any
	}{
		{"nvmlInit_v2", &n.init},
		{"nvmlDeviceGetCount_v2", &n.count},
		{"nvmlDeviceGetHandleByIndex_v2", &n.handle},
		{"nvmlDeviceGetName", &n.name},
		{"nvmlDeviceGetUtilizationRates", &n.util},
		{"nvmlDeviceGetTemperature", &n.temp},
		{"nvmlDeviceGetMemoryInfo", &n.mem},
	}
	for _, s := range syms {
		addr, err := purego.Dlsym(lib, s.name)
		if err != nil {
			return nil, fmt.Errorf("nvml symbol %s: %w", s.name, err)
		}
		purego.RegisterFunc(s.fn, addr)
	}
	if r := n.init(); r != 0 {
		return nil, fmt.Errorf("nvmlInit_v2: %d", r)
	}
	return n, nil
}

func (n *nvmlLib) firstGPU() (GPUMetrics, error) {
	var count uint32
	if r := n.count(&count); r != 0 || count == 0 {
		return GPUMetrics{}, fmt.Errorf("no nvml devices")
	}
	var dev unsafe.Pointer
	if r := n.handle(0, &dev); r != 0 {
		return GPUMetrics{}, fmt.Errorf("nvml handle: %d", r)
	}
	nameBuf := make([]byte, 96)
	if r := n.name(dev, &nameBuf[0], uint32(len(nameBuf))); r != 0 {
		return GPUMetrics{}, fmt.Errorf("nvml name: %d", r)
	}
	var m GPUMetrics
	m.Vendor = "NVIDIA"
	m.Model = strings.TrimRight(string(nameBuf), "\x00")
	var u nvmlUtil
	if r := n.util(dev, &u); r == 0 {
		m.Usage = float64(u.Gpu)
	}
	var temp uint32
	if r := n.temp(dev, 0, &temp); r == 0 { // NVML_TEMPERATURE_GPU
		f := float64(temp)
		m.Temperature = &f
	}
	var mem nvmlMem
	if r := n.mem(dev, &mem); r == 0 {
		m.VRAMTotal, m.VRAMUsed = mem.Total, mem.Used
	}
	return m, nil
}

// --- nvidia-smi fallback ---

// parseSmiCSV parses one "name, util %, temp, mem.used MiB, mem.total MiB" line.
func parseSmiCSV(out string) (GPUMetrics, error) {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	f := strings.Split(line, ",")
	if len(f) != 5 {
		return GPUMetrics{}, fmt.Errorf("unexpected nvidia-smi output: %q", line)
	}
	for i := range f {
		f[i] = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(f[i]), "%"))
	}
	atoi := func(s string) (uint64, error) {
		return strconv.ParseUint(strings.Fields(s)[0], 10, 64)
	}
	util, err := atoi(f[1])
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("bad smi util: %w", err)
	}
	temp, err := atoi(f[2])
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("bad smi temp: %w", err)
	}
	usedMiB, err := atoi(f[3])
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("bad smi mem used: %w", err)
	}
	totalMiB, err := atoi(f[4])
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("bad smi mem total: %w", err)
	}
	t := float64(temp)
	return GPUMetrics{
		Vendor: "NVIDIA", Model: f[0], Usage: float64(util), Temperature: &t,
		VRAMUsed: usedMiB << 20, VRAMTotal: totalMiB << 20,
	}, nil
}

// --- AMD sysfs fallback ---

func amdGPU() (GPUMetrics, error) {
	entries, err := os.ReadDir(drmRoot)
	if err != nil {
		return GPUMetrics{}, err
	}
	for _, e := range entries {
		dev := filepath.Join(drmRoot, e.Name(), "device")
		busyRaw, err := readFile(filepath.Join(dev, "gpu_busy_percent"))
		if err != nil {
			continue
		}
		busy, err := strconv.ParseFloat(strings.TrimSpace(string(busyRaw)), 64)
		if err != nil {
			continue
		}
		var m GPUMetrics
		m.Vendor, m.Model, m.Usage = "AMD", e.Name(), busy
		if t, err := amdTemp(); err == nil {
			m.Temperature = &t
		}
		m.VRAMUsed = readUintOrZero(filepath.Join(dev, "mem_info_vram_used"))
		m.VRAMTotal = readUintOrZero(filepath.Join(dev, "mem_info_vram_total"))
		return m, nil
	}
	return GPUMetrics{}, fmt.Errorf("no AMD GPU found")
}

func amdTemp() (float64, error) {
	entries, err := os.ReadDir(hwmonRoot)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		nameRaw, err := readFile(filepath.Join(hwmonRoot, e.Name(), "name"))
		if err != nil || strings.TrimSpace(string(nameRaw)) != "amdgpu" {
			continue
		}
		return readHwmonTemp(e.Name())
	}
	return 0, fmt.Errorf("no amdgpu sensor")
}

func readUintOrZero(path string) uint64 {
	raw, err := readFile(path)
	if err != nil {
		return 0
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
