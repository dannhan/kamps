package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWorkspaceInitPrintsTemplateWithoutWriting(t *testing.T) {
	opts, err := parseWorkspaceInit([]string{"--lms-id", "17964", "--shortname", "MIK1624303", "--path", "./basdat"})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := printWorkspaceTemplate(&out, opts); err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Version int `json:"version"`
		Courses []struct {
			LMSID     int    `json:"lms_id"`
			Shortname string `json:"shortname"`
			Path      string `json:"path"`
		} `json:"courses"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Version != 1 || len(parsed.Courses) != 1 {
		t.Fatalf("template = %+v", parsed)
	}
	if parsed.Courses[0].LMSID != 17964 || parsed.Courses[0].Path != "./basdat" {
		t.Fatalf("course = %+v", parsed.Courses[0])
	}
	if strings.Contains(out.String(), "MoodleSession") {
		t.Fatal("template should not contain credentials")
	}
}

func TestWorkspaceInitRejectsUnknownOption(t *testing.T) {
	if _, err := parseWorkspaceInit([]string{"--nope"}); err == nil {
		t.Fatal("expected an error")
	}
}
