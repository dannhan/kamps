package source

import (
	"context"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

// Demo is an in-memory source with a realistic-looking dataset. It exists so
// the UI can be built and enjoyed before any networking is written.
//
// Due dates are relative to "now" so the view always has something due soon,
// something overdue, and something far away.
type Demo struct {
	courses     []model.Course
	assignments []model.Assignment
	materials   []model.Material
}

// NewDemo builds the demo dataset.
func NewDemo() *Demo {
	now := time.Now()
	day := func(d int) time.Time {
		t := now.AddDate(0, 0, d)
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 55, 0, 0, t.Location())
	}

	courses := []model.Course{
		{ID: 17964, Code: "MIK1624303", Name: "Basis Data C", Term: "smt11", Source: "lms"},
		{ID: 17638, Code: "MIK1624302", Name: "Struktur Data B", Term: "smt11", Source: "lms",
			HasPraktikum: true, PraktikumURL: "drive.google.com/drive/folders/…strukdat"},
		{ID: 18515, Code: "PAIK6504", Name: "Proyek Perangkat Lunak A", Term: "smt11", Source: "lms"},
		{ID: 18326, Code: "PAIK6304", Name: "Metode Numerik A", Term: "smt11", Source: "lms",
			HasPraktikum: true},
		{ID: 0, Code: "—", Name: "IMK", Term: "smt11", Source: "whatsapp"},
	}

	assignments := []model.Assignment{
		{ID: "K165483", CourseID: 17638, CourseCode: "MIK1624302", Name: "Submission Praktikum 6", Due: day(4), HasDue: true, Status: model.Todo, Praktikum: true, Attachments: 2},
		{ID: "K165491", CourseID: 17638, CourseCode: "MIK1624302", Name: "Quiz Struktur Data", Due: day(9), HasDue: true, Status: model.Todo},
		{ID: "K152742", CourseID: 18326, CourseCode: "PAIK6304", Name: "Tugas Individu VIII", Due: day(13), HasDue: true, Status: model.Todo},
		{ID: "K172325", CourseID: 18515, CourseCode: "PAIK6504", Name: "Progress Report 6", Due: day(12), HasDue: true, Status: model.Doing, Attachments: 1},
		{ID: "K163827", CourseID: 18515, CourseCode: "PAIK6504", Name: "Progress Report 5 (UTS)", Due: day(-3), HasDue: true, Status: model.Todo},
		{ID: "K172331", CourseID: 18515, CourseCode: "PAIK6504", Name: "Progress report 9", Due: day(1), HasDue: true, Status: model.Todo},
		{ID: "K167686", CourseID: 18326, CourseCode: "PAIK6304", Name: "Proyek Metode Numerik", Due: day(6), HasDue: true, Status: model.Doing},
		{ID: "K152704", CourseID: 18326, CourseCode: "PAIK6304", Name: "Tugas Kelompok I. Galat", Due: day(-45), HasDue: true, Status: model.Done},
		{ID: "K152759", CourseID: 18326, CourseCode: "PAIK6304", Name: "QUIZ SPNL", Due: day(54), HasDue: true, Status: model.Todo},
		{ID: "K160843", CourseID: 17964, CourseCode: "MIK1624303", Name: "Quiz Basis Data 1", Due: day(20), HasDue: true, Status: model.Todo},
		{ID: "K155170", CourseID: 17638, CourseCode: "MIK1624302", Name: "Submission Praktikum 1", Due: day(-48), HasDue: true, Status: model.Done, Praktikum: true},
		{ID: "K161202", CourseID: 17638, CourseCode: "MIK1624302", Name: "Tugas Individu Struktur Data", HasDue: false, Status: model.Todo},
	}

	materials := []model.Material{
		{ID: "M165401", CourseID: 17638, CourseCode: "MIK1624302", Name: "0. Struktur Data.pptx", Section: "week-01", Kind: "file", Size: 4404019, Downloaded: true},
		{ID: "M165402", CourseID: 17638, CourseCode: "MIK1624302", Name: "Modul Praktikum 6.pdf", Section: "week-06", Kind: "file", Size: 1153433},
		{ID: "M165403", CourseID: 17638, CourseCode: "MIK1624302", Name: "contoh-kode-6.zip", Section: "week-06", Kind: "file", Size: 839680},
		{ID: "M165404", CourseID: 17638, CourseCode: "MIK1624302", Name: "Slide Pertemuan 5.pdf", Section: "week-05", Kind: "file", Size: 2202009},
		{ID: "M179001", CourseID: 17964, CourseCode: "MIK1624303", Name: "Normalisasi 1-2-3.ppt", Section: "week-05", Kind: "file", Size: 3145728, Downloaded: true},
		{ID: "M179002", CourseID: 17964, CourseCode: "MIK1624303", Name: "Latihan SQL.pdf", Section: "week-06", Kind: "file", Size: 524288},
		{ID: "M185001", CourseID: 18515, CourseCode: "PAIK6504", Name: "Template Progress Report.docx", Section: "week-01", Kind: "file", Size: 45234, Downloaded: true},
		{ID: "M185002", CourseID: 18515, CourseCode: "PAIK6504", Name: "Panduan Sprint 2.pdf", Section: "week-07", Kind: "file", Size: 2097152},
		{ID: "M183001", CourseID: 18326, CourseCode: "PAIK6304", Name: "Galat dan Deret Taylor.pdf", Section: "week-02", Kind: "file", Size: 1572864, Downloaded: true},
		{ID: "M183002", CourseID: 18326, CourseCode: "PAIK6304", Name: "Metode Biseksi.pdf", Section: "week-03", Kind: "file", Size: 1048576},
		{ID: "M183003", CourseID: 18326, CourseCode: "PAIK6304", Name: "Tugas Proyek - spesifikasi.pdf", Section: "week-06", Kind: "file", Size: 786432},
		{ID: "M183004", CourseID: 18326, CourseCode: "PAIK6304", Name: "Referensi online (Moodle)", Section: "week-04", Kind: "url", URL: "https://contoh.example/ref"},
		{ID: "M165483:modul6.pdf", CourseID: 17638, CourseCode: "MIK1624302", Name: "modul6.pdf", Section: "week-06", Kind: "attachment", Size: 622592},
		{ID: "M165483:starter.zip", CourseID: 17638, CourseCode: "MIK1624302", Name: "starter.zip", Section: "week-06", Kind: "attachment", Size: 1048576, Downloaded: true},
		{ID: "PDRIVE-1", CourseID: 17638, CourseCode: "MIK1624302", Name: "Drive: Session 7 slides", Kind: "praktikum", URL: "drive.google.com/…"},
		{ID: "PDRIVE-2", CourseID: 17638, CourseCode: "MIK1624302", Name: "Drive: Session 7 coding demo", Kind: "praktikum", URL: "drive.google.com/…"},
	}

	return &Demo{courses: courses, assignments: assignments, materials: materials}
}

func (d *Demo) Label() string { return "demo (no network)" }

func (d *Demo) Courses(ctx context.Context) ([]model.Course, error) {
	return append([]model.Course(nil), d.courses...), nil
}

func (d *Demo) Assignments(ctx context.Context) ([]model.Assignment, error) {
	return append([]model.Assignment(nil), d.assignments...), nil
}

func (d *Demo) Materials(ctx context.Context) ([]model.Material, error) {
	return append([]model.Material(nil), d.materials...), nil
}

var _ Source = (*Demo)(nil)
