package kulon2

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("legacy fixture unavailable: %v", err)
	}
	return body
}

func TestParseCoursePageLegacyFixture(t *testing.T) {
	sections, modules, err := ParseCoursePage(fixture(t, "course_view_17638.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 2 || sections[0].ID != "5" || sections[0].Name != "Week 1 — 31 August - 6 September" {
		t.Fatalf("sections = %+v", sections)
	}
	var types []string
	for _, module := range modules {
		types = append(types, module.Type)
	}
	if !reflect.DeepEqual(types, []string{"resource", "assign", "assign", "resource", "assign"}) {
		t.Fatalf("module types = %v", types)
	}
	if modules[0].CMID != "165401" || modules[0].Section != "5" {
		t.Fatalf("first module = %+v", modules[0])
	}
}

func TestParseAssignmentPageDateAndAttachments(t *testing.T) {
	page, err := ParseAssignmentPage(fixture(t, "assign_155170.html"), "155170")
	if err != nil {
		t.Fatal(err)
	}
	if page.DueISO != "2026-08-31T23:55:00" || page.DueRaw != "Monday, 31 August 2026, 11:55 PM" || page.LMSStatus != "unknown" {
		t.Fatalf("assignment page = %+v", page)
	}
	attachments, err := ParseAssignmentPage(fixture(t, "assign_163834.html"), "163834")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, attachment := range attachments.Attachments {
		ids = append(ids, attachment.ID)
	}
	if !reflect.DeepEqual(ids, []string{"M163834:laporan.pdf", "M163834:laporan.pdf:2"}) {
		t.Fatalf("attachment IDs = %v", ids)
	}
}

func TestMapCourseModules(t *testing.T) {
	course := model.Course{ID: 17638, Short: "MIK1624302"}
	assignment, material := MapModule(course, Module{Type: "assign", CMID: "155170", Name: "Assignment", Section: "5"})
	if assignment == nil || assignment.ID != "K155170" || material != nil {
		t.Fatalf("assignment=%+v material=%+v", assignment, material)
	}
	assignment, material = MapModule(course, Module{Type: "resource", CMID: "165401", Name: "Slides", Section: "5"})
	if material == nil || material.ID != "M165401" || material.Section != "5" || assignment != nil {
		t.Fatalf("assignment=%+v material=%+v", assignment, material)
	}
	if a, m := MapModule(course, Module{Type: "forum", CMID: "1"}); a != nil || m != nil {
		t.Fatalf("unknown module mapped: %v %v", a, m)
	}
}
