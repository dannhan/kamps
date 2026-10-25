package source

import (
	"fmt"
	"sort"
	"sync"
)

// Options are the dependencies a source factory may need.
type Options struct {
	SiteURL     string
	SessionPath string
	StateDir    string
	LibraryDir  string
}

// Factory builds a source from configuration.
type Factory func(Options) (Source, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register adds an in-tree source under a stable name.
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Open builds the named source.
func Open(name string, opts Options) (Source, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown source %q (available: %v)", name, Names())
	}
	return factory(opts)
}

// Names returns the registered source names in sorted order.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Capabilities describes what a source can provide.
type Capabilities struct {
	Assignments bool
	Materials   bool
	Downloads   bool
	Session     bool
}

// Capable is implemented by sources that want to advertise their capabilities.
type Capable interface {
	Capabilities() Capabilities
}
