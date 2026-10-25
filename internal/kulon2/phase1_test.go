package kulon2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/session"
)

func TestParseFolderPage(t *testing.T) {
	page := []byte(`<html><body>
<a href="/pluginfile.php/1/mod_folder/content/0/a.pdf"><span class="filename">a.pdf</span></a>
<a href="/pluginfile.php/1/mod_folder/content/0/b.pdf"><span class="filename">b.pdf</span></a>
<a href="/pluginfile.php/1/mod_folder/content/0/a.pdf">duplicate</a>
</body></html>`)
	files := ParseFolderPage(page, "42")
	if len(files) != 2 {
		t.Fatalf("files = %+v", files)
	}
	if files[0].ID != "M42:a.pdf" || files[1].ID != "M42:b.pdf" {
		t.Fatalf("ids = %+v", files)
	}
}

func TestLiveSourceMaterializesAttachmentsFoldersAndSubmission(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[{"data":[{"id":1,"shortname":"MIK1","fullname":"Course One"}]}]`)
	})
	mux.HandleFunc("/course/view.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<li id="section-5"><h3 class="sectionname">Week 1</h3><ul>
<li class="activity assign modtype_assign"><a href="/mod/assign/view.php?id=10"><span class="instancename">Essay</span></a></li>
<li class="activity folder modtype_folder"><a href="/mod/folder/view.php?id=20"><span class="instancename">Slides</span></a></li>
</ul></li>`)
	})
	mux.HandleFunc("/mod/assign/view.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>Submitted for grading
<a href="/pluginfile.php/1/mod_assign/intro/att.pdf"><span class="filename">att.pdf</span></a>
</body></html>`)
	})
	mux.HandleFunc("/mod/folder/view.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><a href="/pluginfile.php/1/mod_folder/content/0/week1.pdf"><span class="filename">week1.pdf</span></a></body></html>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie", SessKey: "saved-key"})
	snapshot, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Assignments) != 1 {
		t.Fatalf("assignments = %+v", snapshot.Assignments)
	}
	assignment := snapshot.Assignments[0]
	if assignment.Submission != model.SubmissionSubmitted || assignment.Attachments != 1 || assignment.Section != "Week 1" {
		t.Fatalf("assignment = %+v", assignment)
	}
	if len(snapshot.Materials) != 2 {
		t.Fatalf("materials = %+v", snapshot.Materials)
	}
	var sawAttachment, sawFolder bool
	for _, material := range snapshot.Materials {
		switch material.Kind {
		case "attachment":
			sawAttachment = true
			if material.AssignmentID != "K10" || material.URL == "" || material.Section != "Week 1" {
				t.Fatalf("attachment = %+v", material)
			}
		case "folder-file":
			sawFolder = true
			if material.Folder != "Slides" || material.ParentID != "M20" {
				t.Fatalf("folder file = %+v", material)
			}
		}
	}
	if !sawAttachment || !sawFolder {
		t.Fatalf("missing material kinds: %+v", snapshot.Materials)
	}
	if strings.Contains(snapshot.Notice, "not loaded yet") {
		t.Fatalf("stale notice = %q", snapshot.Notice)
	}
}

func TestParseQuizPageAndMapPageModule(t *testing.T) {
	page := []byte(`<html><body>
<div data-region="activity-dates"><strong>Due:</strong> Monday, 31 August 2026, 11:55 PM</div>
Submitted for grading
<a href="/pluginfile.php/1/mod_quiz/intro/guide.pdf"><span class="filename">guide.pdf</span></a>
</body></html>`)
	parsed, err := ParseQuizPage(page, "77")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.LMSStatus != "submitted" || parsed.DueISO != "2026-08-31T23:55:00" {
		t.Fatalf("quiz page = %+v", parsed)
	}
	if len(parsed.Attachments) != 1 || parsed.Attachments[0].ID != "M77:guide.pdf" {
		t.Fatalf("quiz attachments = %+v", parsed.Attachments)
	}

	course := model.Course{ID: 1, Short: "MIK1"}
	assignment, material := MapModule(course, Module{Type: "page", CMID: "88", Name: "Reading", Section: "Week 1", URL: "/mod/page/view.php?id=88"}, "https://lms.test")
	if assignment != nil || material == nil || material.Kind != "page" {
		t.Fatalf("page mapping: assignment=%+v material=%+v", assignment, material)
	}
}

func TestLiveSourceLoadsQuizDeadline(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[{"data":[{"id":1,"shortname":"MIK1","fullname":"Course One"}]}]`)
	})
	mux.HandleFunc("/course/view.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<li id="section-5"><h3 class="sectionname">Week 1</h3><ul>
<li class="activity quiz modtype_quiz"><a href="/mod/quiz/view.php?id=30"><span class="instancename">Quiz 1</span></a></li>
</ul></li>`)
	})
	mux.HandleFunc("/mod/quiz/view.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><div data-region="activity-dates"><strong>Due:</strong> Monday, 31 August 2026, 11:55 PM</div>Not submitted</body></html>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie", SessKey: "saved-key"})
	snapshot, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Assignments) != 1 {
		t.Fatalf("assignments = %+v", snapshot.Assignments)
	}
	assignment := snapshot.Assignments[0]
	if !assignment.HasDue || assignment.Due.Format("2006-01-02 15:04") != "2026-08-31 23:55" {
		t.Fatalf("quiz due = %+v", assignment)
	}
	if assignment.Submission != model.SubmissionNotSubmitted {
		t.Fatalf("submission = %v", assignment.Submission)
	}
}
