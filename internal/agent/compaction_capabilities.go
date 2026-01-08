package agent

// CompactionCapabilities captures optional provider compaction features.
type CompactionCapabilities struct {
	Remote bool
}

// CompactionCapableProvider exposes provider compaction feature flags.
type CompactionCapableProvider interface {
	Provider
	CompactionCapabilities() CompactionCapabilities
}

// SupportsRemoteCompaction reports whether the provider supports remote compaction.
func SupportsRemoteCompaction(provider Provider) bool {
	capable, ok := provider.(CompactionCapableProvider)
	if !ok {
		return false
	}
	return capable.CompactionCapabilities().Remote
}
