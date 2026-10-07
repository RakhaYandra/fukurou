package collectors

// DefaultRegistry builds the Phase 1 collector set, keeping only modules
// for which enabled reports true. Shared by the --debug path and the GTK
// dashboard so the module list is defined exactly once.
func DefaultRegistry(enabled func(string) bool) *Registry {
	r := NewRegistry()
	all := map[string]Collector{
		"system":  &SystemCollector{},
		"cpu":     &CPUCollector{},
		"memory":  &MemoryCollector{},
		"gpu":     &GPUCollector{},
		"storage": &StorageCollector{},
		"network": &NetworkCollector{},
		"battery": &BatteryCollector{},
	}
	for name, c := range all {
		if enabled(name) {
			_ = r.Register(c) // names unique by construction
		}
	}
	return r
}
