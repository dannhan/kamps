package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type workspaceInitOptions struct {
	LMSID     int
	Shortname string
	Path      string
}

func parseWorkspaceInit(args []string) (workspaceInitOptions, error) {
	opts := workspaceInitOptions{Shortname: "COURSE-SHORTNAME", Path: "."}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--lms-id":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--lms-id needs a value")
			}
			i++
			id, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("--lms-id must be a number")
			}
			opts.LMSID = id
		case strings.HasPrefix(arg, "--lms-id="):
			id, err := strconv.Atoi(strings.TrimPrefix(arg, "--lms-id="))
			if err != nil {
				return opts, fmt.Errorf("--lms-id must be a number")
			}
			opts.LMSID = id
		case arg == "--shortname":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--shortname needs a value")
			}
			i++
			opts.Shortname = args[i]
		case strings.HasPrefix(arg, "--shortname="):
			opts.Shortname = strings.TrimPrefix(arg, "--shortname=")
		case arg == "--path":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--path needs a value")
			}
			i++
			opts.Path = args[i]
		case strings.HasPrefix(arg, "--path="):
			opts.Path = strings.TrimPrefix(arg, "--path=")
		default:
			return opts, fmt.Errorf("unknown workspace init option %q", arg)
		}
	}
	if opts.Path == "" {
		opts.Path = "."
	}
	if opts.Shortname == "" {
		opts.Shortname = "COURSE-SHORTNAME"
	}
	return opts, nil
}

// printWorkspaceTemplate prints a starter course.json. It never writes a file.
func printWorkspaceTemplate(w io.Writer, opts workspaceInitOptions) error {
	template := struct {
		Version int `json:"version"`
		Courses []struct {
			LMSID     int    `json:"lms_id"`
			Shortname string `json:"shortname"`
			Path      string `json:"path"`
		} `json:"courses"`
	}{Version: 1}
	template.Courses = append(template.Courses, struct {
		LMSID     int    `json:"lms_id"`
		Shortname string `json:"shortname"`
		Path      string `json:"path"`
	}{LMSID: opts.LMSID, Shortname: opts.Shortname, Path: opts.Path})

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(template); err != nil {
		return err
	}
	return nil
}
