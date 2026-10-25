package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/api"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/core"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

// runHeadless handles JSON/scriptable commands before the TUI is considered.
// The bool reports whether the command was handled; the int is the exit code.
func runHeadless(args []string, cfg config.Config, src source.Source, stdout, stderr io.Writer) (bool, int) {
	args, jsonOut := stripJSON(args)
	if len(args) == 0 {
		return false, 0
	}
	command := args[0]
	args = args[1:]
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		printCommandHelp(stdout, command)
		return true, 0
	}
	switch command {
	case "courses":
		return true, headlessRead(command, args, jsonOut, cfg, src, stdout, stderr, func(service *core.Service, _ headlessQuery) any {
			return api.Courses(service.Snapshot().Courses)
		}, func(service *core.Service, _ headlessQuery) (string, error) {
			var b strings.Builder
			for _, course := range service.Snapshot().Courses {
				fmt.Fprintf(&b, "%d\t%s\t%s\t%s\n", course.ID, course.Code, course.Name, course.Term)
			}
			return b.String(), nil
		})
	case "tasks":
		return true, headlessRead(command, args, jsonOut, cfg, src, stdout, stderr, func(service *core.Service, query headlessQuery) any {
			return api.Tasks(service.Tasks(core.Query{Filter: query.filter, Search: query.search, CourseID: query.courseID}), time.Now())
		}, func(service *core.Service, query headlessQuery) (string, error) {
			var b strings.Builder
			for _, task := range service.Tasks(core.Query{Filter: query.filter, Search: query.search, CourseID: query.courseID}) {
				due := "no due"
				if task.HasDue {
					due = task.Due.Format("2006-01-02")
				}
				fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", task.ID, task.Triage, due, task.CourseCode, task.Name)
			}
			return b.String(), nil
		})
	case "materials":
		return true, headlessRead(command, args, jsonOut, cfg, src, stdout, stderr, func(service *core.Service, query headlessQuery) any {
			return api.Materials(service.Materials(core.Query{Filter: query.filter, Search: query.search, CourseID: query.courseID}))
		}, func(service *core.Service, query headlessQuery) (string, error) {
			var b strings.Builder
			for _, material := range service.Materials(core.Query{Filter: query.filter, Search: query.search, CourseID: query.courseID}) {
				state := "new"
				if material.Downloaded {
					state = "local"
				}
				fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", material.ID, state, core.SectionLabel(material.Section), material.Name)
			}
			return b.String(), nil
		})
	case "status":
		return true, headlessStatus(command, args, jsonOut, cfg, src, stdout, stderr)
	case "sync":
		return true, headlessStatus(command, args, jsonOut, cfg, src, stdout, stderr)
	case "triage":
		return true, headlessTriage(args, jsonOut, cfg, src, stdout, stderr)
	case "download":
		return true, headlessDownload(args, jsonOut, cfg, src, stdout, stderr)
	case "export":
		return true, headlessExport(args, jsonOut, cfg, src, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "kamps: unknown command %q\n", command)
		return true, 2
	}
}

type headlessQuery struct {
	filter   string
	search   string
	course   string
	courseID int
}

func headlessRead(command string, args []string, jsonOut bool, cfg config.Config, src source.Source, stdout, stderr io.Writer, jsonItems func(*core.Service, headlessQuery) any, textItems func(*core.Service, headlessQuery) (string, error)) int {
	query, err := parseQueryArgs(args)
	if err != nil {
		return headlessError(stderr, jsonOut, "usage", err)
	}
	if !validFilter(command, query.filter) {
		return headlessError(stderr, jsonOut, "usage", fmt.Errorf("unknown filter %q", query.filter))
	}
	service, _, err := loadHeadless(cfg, src)
	if err != nil {
		return headlessError(stderr, jsonOut, "source", err)
	}
	if query.course != "" {
		course, ok := service.Course(query.course)
		if !ok {
			return headlessError(stderr, jsonOut, "usage", fmt.Errorf("unknown course %q", query.course))
		}
		query.courseID = course.ID
	}
	if jsonOut {
		if err := api.Write(stdout, command, jsonItems(service, query), nil); err != nil {
			return 1
		}
		return 0
	}
	text, err := textItems(service, query)
	if err != nil {
		return headlessError(stderr, jsonOut, "export", err)
	}
	_, _ = io.WriteString(stdout, text)
	return 0
}

func headlessStatus(command string, args []string, jsonOut bool, cfg config.Config, src source.Source, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		return headlessError(stderr, jsonOut, "usage", fmt.Errorf("%s takes no arguments", command))
	}
	service, snapshot, err := loadHeadless(cfg, src)
	if err != nil {
		return headlessError(stderr, jsonOut, "source", err)
	}
	status := service.Status()
	status.FromCache = snapshot.FromCache
	if !snapshot.CachedAt.IsZero() {
		value := snapshot.CachedAt
		status.CachedAt = &value
	}
	if jsonOut {
		if err := api.Write(stdout, command, status, nil); err != nil {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "source: %s\n", status.Source)
	fmt.Fprintf(stdout, "session: ok=%t", status.SessionOK)
	if status.SessionExpiresAt != nil {
		fmt.Fprintf(stdout, " expires=%s", status.SessionExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	fmt.Fprintln(stdout)
	if status.LastSync != nil {
		fmt.Fprintf(stdout, "last sync: %s\n", status.LastSync.Format("2006-01-02T15:04:05Z07:00"))
	}
	if status.FromCache && status.CachedAt != nil {
		fmt.Fprintf(stdout, "cache: %s\n", status.CachedAt.Format("2006-01-02 15:04"))
	}
	fmt.Fprintf(stdout, "courses: %d\nassignments: %d\nmaterials: %d\ndownloaded: %d\n", status.Courses, status.Assignments, status.Materials, status.Downloaded)
	return 0
}

func headlessTriage(args []string, jsonOut bool, cfg config.Config, src source.Source, stdout, stderr io.Writer) int {
	ids, decision, err := parseTriageArgs(args)
	if err != nil {
		return headlessError(stderr, jsonOut, "usage", err)
	}
	service, _, err := loadHeadless(cfg, src)
	if err != nil {
		return headlessError(stderr, jsonOut, "source", err)
	}
	changed, err := service.Decide(ids, decision)
	if err != nil {
		return headlessError(stderr, jsonOut, "state", err)
	}
	if jsonOut {
		items := make([]api.Decision, 0, len(changed))
		for _, item := range changed {
			items = append(items, api.Decision{ID: item.ID, Triage: item.Triage.String()})
		}
		if err := api.Write(stdout, "triage", items, nil); err != nil {
			return 1
		}
		return 0
	}
	for _, item := range changed {
		fmt.Fprintf(stdout, "%s\t%s\n", item.ID, item.Triage)
	}
	return 0
}

func headlessDownload(args []string, jsonOut bool, cfg config.Config, src source.Source, stdout, stderr io.Writer) int {
	opts, err := parseDownloadArgs(args)
	if err != nil {
		return headlessError(stderr, jsonOut, "usage", err)
	}
	service, _, err := loadHeadless(cfg, src)
	if err != nil {
		return headlessError(stderr, jsonOut, "source", err)
	}
	ids, targets, err := downloadTargets(service, opts)
	if err != nil {
		return headlessError(stderr, jsonOut, "usage", err)
	}
	if opts.dryRun {
		if jsonOut {
			if err := api.Write(stdout, "download", targets, nil); err != nil {
				return 1
			}
			return 0
		}
		for _, target := range targets {
			fmt.Fprintf(stdout, "%s\t%s\n", target.ID, target.Path)
		}
		return 0
	}
	results, err := service.Download(context.Background(), ids, func(id string, done, total int64) {
		if !jsonOut {
			fmt.Fprintf(stderr, "%s\t%d/%d\n", id, done, total)
		}
	})
	if err != nil {
		if jsonOut {
			_ = api.WriteError(stderr, "download", err.Error())
			return 4
		}
		return headlessError(stderr, false, "download", err)
	}
	if jsonOut {
		items := make([]api.Download, 0, len(results))
		for _, result := range results {
			items = append(items, api.Download{ID: result.ID, Path: result.Path, Bytes: result.Bytes})
		}
		if err := api.Write(stdout, "download", items, nil); err != nil {
			return 1
		}
		return 0
	}
	for _, result := range results {
		fmt.Fprintf(stdout, "%s\t%s\n", result.ID, result.Path)
	}
	return 0
}

func headlessExport(args []string, jsonOut bool, cfg config.Config, src source.Source, stdout, stderr io.Writer) int {
	format, out, ids, err := parseExportArgs(args)
	if err != nil {
		return headlessError(stderr, jsonOut, "usage", err)
	}
	service, _, err := loadHeadless(cfg, src)
	if err != nil {
		return headlessError(stderr, jsonOut, "source", err)
	}
	result, err := service.Export(format, out, ids)
	if err != nil {
		return headlessError(stderr, jsonOut, "export", err)
	}
	if jsonOut {
		items := api.ExportResult{Added: result.Added, Skipped: result.Skipped, Path: result.Path}
		if err := api.Write(stdout, "export", items, nil); err != nil {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "wrote %d, skipped %d, %s\n", result.Added, result.Skipped, result.Path)
	return 0
}

func loadHeadless(cfg config.Config, src source.Source) (*core.Service, core.Snapshot, error) {
	service := core.New(src, cfg)
	snapshot, err := service.Load(context.Background())
	if err != nil {
		if cached, ok := service.LoadCached(); ok {
			return service, cached, nil
		}
		return service, core.Snapshot{}, err
	}
	return service, snapshot, nil
}

func headlessError(stderr io.Writer, jsonOut bool, code string, err error) int {
	exit := 2
	switch code {
	case "auth":
		exit = 3
	case "source", "download":
		exit = 4
	case "state":
		exit = 5
	case "export":
		exit = 1
	}
	if jsonOut {
		_ = api.WriteError(stderr, code, err.Error())
		return exit
	}
	fmt.Fprintln(stderr, "kamps:", err)
	return exit
}

func stripJSON(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			continue
		}
		out = append(out, arg)
	}
	return out, jsonOut
}

func validFilter(command, filter string) bool {
	if filter == "" || filter == "all" {
		return true
	}
	switch command {
	case "tasks":
		return filter == "undecided" || filter == "new" || filter == "overdue" || filter == "tracked" || filter == "missed" || filter == "ignored"
	case "materials":
		return filter == "new" || filter == "downloaded" || filter == "local" || filter == "link"
	}
	return false
}

func parseQueryArgs(args []string) (headlessQuery, error) {
	var query headlessQuery
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--filter":
			if i+1 >= len(args) {
				return query, fmt.Errorf("--filter needs a value")
			}
			i++
			query.filter = args[i]
		case strings.HasPrefix(arg, "--filter="):
			query.filter = strings.TrimPrefix(arg, "--filter=")
		case arg == "--search":
			if i+1 >= len(args) {
				return query, fmt.Errorf("--search needs a value")
			}
			i++
			query.search = args[i]
		case strings.HasPrefix(arg, "--search="):
			query.search = strings.TrimPrefix(arg, "--search=")
		case arg == "--course":
			if i+1 >= len(args) {
				return query, fmt.Errorf("--course needs a value")
			}
			i++
			query.course = args[i]
		case strings.HasPrefix(arg, "--course="):
			query.course = strings.TrimPrefix(arg, "--course=")
		default:
			return query, fmt.Errorf("unknown argument %q", arg)
		}
	}
	return query, nil
}

func parseTriageArgs(args []string) ([]string, model.Triage, error) {
	var ids []string
	decision := model.Undecided
	chosen := false
	for _, arg := range args {
		var value model.Triage
		switch arg {
		case "--tracked":
			value = model.Tracked
		case "--missed":
			value = model.Missed
		case "--ignored":
			value = model.Ignored
		case "--undecided":
			value = model.Undecided
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, decision, fmt.Errorf("unknown argument %q", arg)
			}
			ids = append(ids, arg)
			continue
		}
		if chosen && value != decision {
			return nil, decision, fmt.Errorf("choose exactly one triage decision")
		}
		decision = value
		chosen = true
	}
	if len(ids) == 0 || !chosen {
		return nil, decision, fmt.Errorf("usage: kamps triage <id...> --tracked|--missed|--ignored|--undecided")
	}
	return ids, decision, nil
}

type downloadOptions struct {
	ids    []string
	allNew bool
	course string
	dryRun bool
}

type downloadTarget struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Downloaded bool   `json:"downloaded"`
}

func parseDownloadArgs(args []string) (downloadOptions, error) {
	var opts downloadOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--all-new":
			opts.allNew = true
		case arg == "--dry-run":
			opts.dryRun = true
		case arg == "--course":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--course needs a value")
			}
			i++
			opts.course = args[i]
		case strings.HasPrefix(arg, "--course="):
			opts.course = strings.TrimPrefix(arg, "--course=")
		default:
			if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("unknown argument %q", arg)
			}
			opts.ids = append(opts.ids, arg)
		}
	}
	if len(opts.ids) == 0 && !opts.allNew && opts.course == "" {
		return opts, fmt.Errorf("choose material ids, --all-new, or --course")
	}
	return opts, nil
}

func downloadTargets(service *core.Service, opts downloadOptions) ([]string, []downloadTarget, error) {
	wanted := map[string]bool{}
	for _, id := range opts.ids {
		wanted[id] = true
	}
	var courseID int
	if opts.course != "" {
		course, ok := service.Course(opts.course)
		if !ok {
			return nil, nil, fmt.Errorf("unknown course %q", opts.course)
		}
		courseID = course.ID
	}
	var ids []string
	var targets []downloadTarget
	for _, material := range service.Snapshot().Materials {
		if len(wanted) > 0 && !wanted[material.ID] {
			continue
		}
		if courseID != 0 && material.CourseID != courseID {
			continue
		}
		if opts.allNew && material.Downloaded {
			continue
		}
		ids = append(ids, material.ID)
		targets = append(targets, downloadTarget{ID: material.ID, Path: service.Destination(material), Downloaded: material.Downloaded})
	}
	if len(ids) == 0 {
		return nil, nil, fmt.Errorf("no matching materials")
	}
	return ids, targets, nil
}

func parseExportArgs(args []string) (format, out string, ids []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("--format needs a value")
			}
			i++
			format = args[i]
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		case arg == "--out":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("--out needs a value")
			}
			i++
			out = args[i]
		case strings.HasPrefix(arg, "--out="):
			out = strings.TrimPrefix(arg, "--out=")
		case arg == "--ids":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("--ids needs a value")
			}
			i++
			ids = splitIDs(args[i])
		case strings.HasPrefix(arg, "--ids="):
			ids = splitIDs(strings.TrimPrefix(arg, "--ids="))
		default:
			return "", "", nil, fmt.Errorf("unknown argument %q", arg)
		}
	}
	return format, out, ids, nil
}

func splitIDs(value string) []string {
	var ids []string
	for _, id := range strings.Split(value, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

var commandUsage = map[string]string{
	"courses":   "usage: kamps courses [--json]",
	"tasks":     "usage: kamps tasks [--json] [--filter all|undecided|overdue|tracked|missed|ignored] [--search Q] [--course ID|CODE]",
	"materials": "usage: kamps materials [--json] [--filter all|new|downloaded|link] [--search Q] [--course ID|CODE]",
	"status":    "usage: kamps status [--json]",
	"sync":      "usage: kamps sync [--json]",
	"triage":    "usage: kamps triage <id...> --tracked|--missed|--ignored|--undecided [--json]",
	"download":  "usage: kamps download <id...> | --all-new | --course ID|CODE [--dry-run] [--json]",
	"export":    "usage: kamps export [--format markdown|todo|taskwarrior] [--out PATH] [--ids A,B] [--json]",
}

func printCommandHelp(w io.Writer, command string) {
	if usage, ok := commandUsage[command]; ok {
		fmt.Fprintln(w, usage)
		return
	}
	fmt.Fprintf(w, "unknown command %q\n", command)
}
