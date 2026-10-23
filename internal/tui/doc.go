// Package tui is the Bubble Tea application: a cheerful terminal UI over a
// source.Source.
//
// The package follows The Elm Architecture:
//
//	Init   -> kick off asynchronous work
//	Update -> react to messages (keys, resize, ticks, loaded data)
//	View   -> render the model to a string
//
// The interactive screen uses Bubble Tea's alternate screen and a borderless
// shell separated by full-width rules.
//
// # File map
//
//	app.go          root model, construction, messages, update, view shell
//	keys.go         help overlay key bindings
//	search.go       live search state, predicates, search-mode keys
//	tree.go         flattened tree node contract and connectors
//	detail.go       split pane and per-node detail content
//	frame.go        title, body, tab cells, rules, footer hint fitting
//	row.go          cursor and selection bands
//	theme.go        semantic colors and reusable styles
//	util.go         width, padding, truncation, wrapping, small helpers
//	plain.go        non-TTY snapshot (also reachable without Bubble Tea)
//	views.go        shared formatting helpers
//
// Feature code is being split out of app.go by vertical slice. When adding a
// key, filter, row, or detail field, start with the tab or feature it belongs
// to, then follow the move-only split documented in docs/code-tour.md.
package tui
