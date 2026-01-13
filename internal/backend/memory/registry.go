package memory

import "cogni/internal/registry"

// AttachRegistry wires a registry to receive applied decrease updates.
func (m *MemoryBackend) AttachRegistry(reg *registry.Registry, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = reg
	m.registryPath = path
}
