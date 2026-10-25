package main

import (
	"fmt"
	"strings"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/export"
)

type options struct {
	noAnim       bool
	demo         bool
	source       string
	commitFile   string
	exportFormat string
}

// parseOptions strips kamps' global flags from args and returns the remaining
// words (command plus its own flags).
func parseOptions(args []string) (options, []string, error) {
	var opts options
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--no-anim" || arg == "--noanim":
			opts.noAnim = true
		case arg == "--demo":
			opts.demo = true
		case arg == "--source":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--source needs a value")
			}
			i++
			opts.source = args[i]
		case strings.HasPrefix(arg, "--source="):
			opts.source = strings.TrimPrefix(arg, "--source=")
		case arg == "--commit-file":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--commit-file needs a path")
			}
			i++
			opts.commitFile = args[i]
		case strings.HasPrefix(arg, "--commit-file="):
			opts.commitFile = strings.TrimPrefix(arg, "--commit-file=")
		case arg == "--export-format":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--export-format needs a value")
			}
			i++
			opts.exportFormat = args[i]
		case strings.HasPrefix(arg, "--export-format="):
			opts.exportFormat = strings.TrimPrefix(arg, "--export-format=")
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func (o options) apply(cfg *config.Config) error {
	cfg.NoAnim = o.noAnim
	if o.source != "" {
		cfg.Source = o.source
	}
	if o.commitFile != "" {
		cfg.CommitFile = o.commitFile
	}
	if o.exportFormat != "" {
		if _, err := export.For(o.exportFormat); err != nil {
			return err
		}
		cfg.ExportFormat = strings.ToLower(o.exportFormat)
	}
	return nil
}

// browserMode parses `--browser`, `--browser=auto|firefox|chromium`.
func browserMode(args []string) (string, bool) {
	for i, arg := range args {
		if arg == "--browser" {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				return args[i+1], true
			}
			return "auto", true
		}
		if strings.HasPrefix(arg, "--browser=") {
			return strings.TrimPrefix(arg, "--browser="), true
		}
	}
	return "", false
}
