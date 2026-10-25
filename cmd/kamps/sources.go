package main

import (
	"github.com/ruyam/kamps/internal/kulon2"
	"github.com/ruyam/kamps/internal/source"
)

func init() {
	source.Register("demo", func(source.Options) (source.Source, error) {
		return source.NewDemo(), nil
	})
	source.Register("kulon2", func(opts source.Options) (source.Source, error) {
		return kulon2.NewSource(opts.SiteURL, opts.SessionPath)
	})
}
