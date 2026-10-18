package source

import (
	"context"
	"fmt"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

// Demo is an in-memory source with a realistic-looking dataset. It exists so
// the UI can be built and enjoyed before any networking is written.
//
// Due dates are relative to "now" so the view always has something overdue,
// something due today and something far away.
type Demo struct {
	courses     []model.Course
	assignments []model.Assignment
	materials   []model.Material
}

// NewDemo builds the demo dataset.
func NewDemo() *Demo {
	now := time.Now()
	at := func(d int) time.Time {
		t := now.AddDate(0, 0, d)
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 55, 0, 0, t.Location())
	}

	courses := []model.Course{
		{ID: 17638, Code: "MIK1624302", Short: "strukdat", Name: "Struktur Data B", Term: "smt11", Source: "lms",
			HasPraktikum: true, PraktikumURL: "drive.google.com/drive/folders/…strukdat"},
		{ID: 18326, Code: "PAIK6304", Short: "metnum", Name: "Metode Numerik A", Term: "smt11", Source: "lms", HasPraktikum: true},
		{ID: 18515, Code: "PAIK6504", Short: "ppl", Name: "Proyek Perangkat Lunak A", Term: "smt11", Source: "lms"},
		{ID: 17964, Code: "MIK1624303", Short: "basdat", Name: "Basis Data C", Term: "smt11", Source: "lms"},
		{ID: 0, Code: "—", Short: "imk", Name: "IMK", Term: "smt11", Source: "whatsapp"},
	}

	// Short course labels are carried on model.Course.Short.
	short := map[int]string{17638: "MIK1624302", 18326: "PAIK6304", 18515: "PAIK6504", 17964: "MIK1624303", 0: "—"}

	var as []model.Assignment
	add := func(id string, cid int, name string, due int, hasDue bool, tr model.Triage, est int, praktikum bool) {
		a := model.Assignment{
			ID: id, CourseID: cid, CourseCode: short[cid], Name: name,
			Triage: tr, EstMinutes: est, Praktikum: praktikum, Goal: "G-college",
		}
		if hasDue {
			a.HasDue = true
			a.Due = at(due)
		}
		as = append(as, a)
	}

	// The hand-picked items shown in the mockups.
	add("K163827", 18515, "Progress Report 5 (UTS)", -3, true, model.Undecided, 90, false)
	add("K152705", 18326, "Tugas Individu I. Galat", -46, true, model.Missed, 60, false)
	add("K152710", 18326, "Tugas Kelompok II. Solusi Numerik", -32, true, model.Tracked, 120, false)
	add("K172331", 18515, "Progress report 9", 1, true, model.Undecided, 75, false)
	add("K167686", 18326, "Proyek Metode Numerik", 6, true, model.Undecided, 240, false)
	add("K152742", 18326, "Tugas Individu VIII", 12, true, model.Undecided, 60, false)
	add("K165483", 17638, "Submission Praktikum 6", 4, true, model.Undecided, 90, true)
	add("K165491", 17638, "Quiz Struktur Data", 9, true, model.Undecided, 30, false)
	add("K160843", 17964, "Quiz Basis Data 1", 20, true, model.Undecided, 30, false)
	add("K152759", 18326, "QUIZ SPNL", 54, true, model.Undecided, 45, false)
	add("K161202", 17638, "Tugas Individu Struktur Data", 0, false, model.Undecided, 60, false)
	add("K164110", 0, "IMK laporan akhir (WA)", 14, true, model.Undecided, 180, false)

	// A backlog of older, still-undecided items — this is the real problem the
	// tool exists to shrink, so the counts should look believable.
	backlog := []struct {
		cid  int
		name string
		due  int
	}{
		{18515, "Progress Report 1", -60}, {18515, "Progress Report 2", -53},
		{18515, "Progress Report 3", -46}, {18515, "Progress Report 4", -39},
		{18326, "Tugas Kelompok I. Galat", -52}, {18326, "Latihan Deret Taylor", -45},
		{18326, "Kuis Biseksi", -38}, {18326, "Tugas Individu III", -31},
		{17638, "Submission Praktikum 1", -55}, {17638, "Submission Praktikum 2", -48},
		{17638, "Submission Praktikum 3", -41}, {17638, "Submission Praktikum 4", -34},
		{17638, "Submission Praktikum 5", -27}, {17964, "Latihan ERD", -50},
		{17964, "Quiz Normalisasi", -44}, {17964, "Tugas SQL Dasar", -37},
		{17964, "Latihan JOIN", -30}, {17964, "Tugas Relasi", -23},
	}
	for i, b := range backlog {
		add(fmt.Sprintf("K9%05d", 1000+i), b.cid, b.name, b.due, true, model.Undecided, 60, false)
	}

	// A couple more already-decided ones, for realistic counts.
	decided := []struct {
		id   string
		cid  int
		name string
		due  int
		tr   model.Triage
	}{
		{"K150001", 17638, "Kuis Struktur Data awal", -58, model.Tracked},
		{"K150002", 18326, "Tugas Kelompok I (lanjut)", -44, model.Tracked},
		{"K150003", 17964, "Latihan Normalisasi", -36, model.Tracked},
		{"K150004", 18515, "Report 0", -62, model.Tracked},
		{"K150005", 0, "IMK pre-test (WA)", -40, model.Tracked},
		{"K150006", 17638, "Contoh praktikum tahun lalu", -57, model.Ignored},
		{"K150007", 18326, "Template laporan", -49, model.Ignored},
		{"K150008", 17964, "Slide rangkuman", -42, model.Ignored},
		{"K150009", 18515, "Dokumen contoh (opsional)", -35, model.Ignored},
		{"K15000A", 18326, "Latihan tambahan (opsional)", -28, model.Ignored},
		{"K15000B", 17638, "Kuis percobaan", -47, model.Missed},
		{"K15000C", 17964, "Tugas tidak dinilai", -33, model.Missed},
	}
	for _, d := range decided {
		add(d.id, d.cid, d.name, d.due, true, d.tr, 60, false)
	}

	materials := []model.Material{
		{ID: "M165401", CourseID: 17638, CourseCode: short[17638], Name: "0. Struktur Data.pptx", Section: "week-01", Kind: "file", Size: 4404019, Downloaded: true},
		{ID: "M165402", CourseID: 17638, CourseCode: short[17638], Name: "Modul Praktikum 6.pdf", Section: "week-06", Kind: "file", Size: 1153433},
		{ID: "M165403", CourseID: 17638, CourseCode: short[17638], Name: "contoh-kode-6.zip", Section: "week-06", Kind: "file", Size: 839680},
		{ID: "M165404", CourseID: 17638, CourseCode: short[17638], Name: "Slide Pertemuan 5.pdf", Section: "week-05", Kind: "file", Size: 2202009},
		{ID: "M179001", CourseID: 17964, CourseCode: short[17964], Name: "Normalisasi 1-2-3.ppt", Section: "week-05", Kind: "file", Size: 3145728, Downloaded: true},
		{ID: "M179002", CourseID: 17964, CourseCode: short[17964], Name: "Latihan SQL.pdf", Section: "week-06", Kind: "file", Size: 524288},
		{ID: "M185001", CourseID: 18515, CourseCode: short[18515], Name: "Template Progress Report.docx", Section: "week-01", Kind: "file", Size: 45234, Downloaded: true},
		{ID: "M185002", CourseID: 18515, CourseCode: short[18515], Name: "Panduan Sprint 2.pdf", Section: "week-07", Kind: "file", Size: 2097152},
		{ID: "M183001", CourseID: 18326, CourseCode: short[18326], Name: "Galat dan Deret Taylor.pdf", Section: "week-02", Kind: "file", Size: 1572864, Downloaded: true},
		{ID: "M183002", CourseID: 18326, CourseCode: short[18326], Name: "Metode Biseksi.pdf", Section: "week-03", Kind: "file", Size: 1048576},
		{ID: "M183003", CourseID: 18326, CourseCode: short[18326], Name: "Tugas Proyek - spesifikasi.pdf", Section: "week-06", Kind: "file", Size: 786432},
		{ID: "M183004", CourseID: 18326, CourseCode: short[18326], Name: "Referensi online (Moodle)", Section: "week-04", Kind: "url", URL: "https://contoh.example/ref"},
		{ID: "M165483:modul6.pdf", CourseID: 17638, CourseCode: short[17638], Name: "modul6.pdf", Section: "week-06", Kind: "attachment", Size: 622592},
		{ID: "M165483:starter.zip", CourseID: 17638, CourseCode: short[17638], Name: "starter.zip", Section: "week-06", Kind: "attachment", Size: 1048576, Downloaded: true},
		{ID: "PDRIVE-1", CourseID: 17638, CourseCode: short[17638], Name: "Drive: Session 7 slides", Kind: "praktikum", URL: "drive.google.com/…"},
		{ID: "PDRIVE-2", CourseID: 17638, CourseCode: short[17638], Name: "Drive: Session 7 coding demo", Kind: "praktikum", URL: "drive.google.com/…"},
	}

	return &Demo{courses: courses, assignments: as, materials: materials}
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
