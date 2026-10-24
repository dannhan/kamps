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
// Root and lifecycle:
//
//	app.go          App struct, construction, messages, update loop
//	state.go        bridge between App and the persisted store
//	keys.go         key bindings and key dispatch
//	navigation.go   tab switching, cursors, folding, selection
//	filters.go      filter names, cycling, clearing
//	search.go       live search state, predicates, search-mode keys
//
// Tasks tab:
//
//	tasks.go        task filtering, target resolution, triage, footer
//	task_tree.go    flattened task tree and grouping
//	task_rows.go    task rows, counts, totals, tab content
//	detail_tasks.go task detail content
//
// Materials tab:
//
//	materials.go      material filtering, tree, downloads, attachment links
//	material_rows.go  material rows and tab content
//	detail_materials.go material detail content
//
// Other tabs and shell:
//
//	commit.go       session queue, preview, clipboard, commit tab
//	sync.go         simulated sync state and tab
//	chrome.go       shell, tab bar, footer, help overlay
//	detail.go       split-pane layout and visibility
//	tree.go         shared tree node contract and connectors
//	frame.go        title, body, tab cells, rules, hint fitting
//	row.go          cursor and selection bands
//	theme.go        semantic colors and reusable styles
//	text.go         width, padding, truncation, wrapping
//	window.go       viewport windows and scroll hints
//	format.go       shared formatting helpers
//	numbers.go      small integer helpers
//	plain.go        non-TTY snapshot (also reachable without Bubble Tea)
//
// Tests live next to the feature they cover: task_tree_test.go,
// task_rows_test.go, materials_tree_test.go, search_test.go, detail_test.go,
// commit_test.go, navigation_test.go, shell_test.go, and store_test.go.
package tui
