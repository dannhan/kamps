package main

import (
	"path/filepath"

	"github.com/ruyam/kamps/internal/kulon2"
	"github.com/ruyam/kamps/internal/source"
)

func init() {
	source.Register("demo", func(source.Options) (source.Source, error) {
		return source.NewDemo(), nil
	})
	source.Register("kulon2", func(opts source.Options) (source.Source, error) {
		lockDir := ""
		if opts.StateDir != "" {
			lockDir = filepath.Join(opts.StateDir, "locks")
		}
		return kulon2.NewSource(opts.SiteURL, opts.SessionPath, lockDir)
	})
}
