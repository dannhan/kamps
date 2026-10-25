package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/core"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

// visibleMaterials returns the Materials-tab files after the active filter and
// search. The tree, counters and cursor all read this one list.
func (m *App) visibleMaterials() []model.Material {
	if m.filter == "unknown" {
		out := make([]model.Material, 0, len(m.unknownFiles))
		for _, material := range m.unknownMaterials() {
			if m.matchesMaterialSearch(material) {
				out = append(out, material)
			}
		}
		return out
	}
	return m.service.Materials(core.Query{Filter: m.filter, Search: m.search})
}

// matNodes is the Materials pane: course ▸ week ▸ folder ▸ file. Weeks keep
// the source's own label; an empty section shows as week-0.
func (m *App) matNodes() []node {
	byCourse := map[int][]model.Material{}
	for _, material := range m.visibleMaterials() {
		byCourse[material.CourseID] = append(byCourse[material.CourseID], material)
	}
	courseIDs := make([]int, 0, len(byCourse))
	for id := range byCourse {
		courseIDs = append(courseIDs, id)
	}
	sort.Ints(courseIDs)
	out := []node{}
	for _, courseID := range courseIDs {
		course := node{kind: nodeCourse, courseID: courseID, key: courseKey(courseID)}
		out = append(out, course)
		if !m.isOpen(course.key) {
			continue
		}

		emitFile := func(file model.Material, fileParents []string) {
			fileNode := node{kind: nodeFile, courseID: courseID, mat: file, key: materialKey(file.ID), parents: fileParents}
			out = append(out, fileNode)
			if assignment := m.assignmentForMaterial(file); assignment != nil && m.isOpen(fileNode.key) {
				appendParents := append(append([]string(nil), fileParents...), fileNode.key)
				out = append(out, node{kind: nodeMaterialAssignment, courseID: courseID, task: *assignment, key: "ma:" + assignment.ID + ":" + file.ID, parents: appendParents})
			}
		}

		bySection := map[string][]model.Material{}
		var sections []string
		for _, file := range byCourse[courseID] {
			label := sectionLabel(file.Section)
			if _, ok := bySection[label]; !ok {
				sections = append(sections, label)
			}
			bySection[label] = append(bySection[label], file)
		}
		sort.Slice(sections, func(i, j int) bool { return sectionLess(sections[i], sections[j]) })
		for _, section := range sections {
			sectionFiles := bySection[section]
			sectionKey := materialSectionKey(courseID, section)
			out = append(out, node{kind: nodeMaterialSection, courseID: courseID, section: section, key: sectionKey, parents: []string{course.key}})
			if !m.isOpen(sectionKey) {
				continue
			}

			byFolder := map[string][]model.Material{}
			var folders []string
			var direct []model.Material
			for _, file := range sectionFiles {
				if file.Folder == "" {
					direct = append(direct, file)
					continue
				}
				if _, ok := byFolder[file.Folder]; !ok {
					folders = append(folders, file.Folder)
				}
				byFolder[file.Folder] = append(byFolder[file.Folder], file)
			}
			sort.Strings(folders)
			for _, folder := range folders {
				folderFiles := byFolder[folder]
				sort.Slice(folderFiles, func(i, j int) bool { return folderFiles[i].Name < folderFiles[j].Name })
				folderKey := materialFolderKey(courseID, section, folder)
				out = append(out, node{kind: nodeMaterialFolder, courseID: courseID, section: section, key: folderKey, parents: []string{course.key, sectionKey}, mat: model.Material{Folder: folder}})
				if !m.isOpen(folderKey) {
					continue
				}
				for _, file := range folderFiles {
					emitFile(file, []string{course.key, sectionKey, folderKey})
				}
			}
			sort.Slice(direct, func(i, j int) bool { return direct[i].Name < direct[j].Name })
			for _, file := range direct {
				emitFile(file, []string{course.key, sectionKey})
			}
		}
	}
	return out
}

// materialNodeFiles lists the materials a row stands for: the file itself,
// every visible file in a week or folder, or every file in a course.
func (m *App) materialNodeFiles(n node) []model.Material {
	switch n.kind {
	case nodeFile:
		return []model.Material{n.mat}
	case nodeMaterialSection:
		files := []model.Material{}
		for _, material := range m.visibleMaterials() {
			if material.CourseID == n.courseID && sectionLabel(material.Section) == n.section {
				files = append(files, material)
			}
		}
		return files
	case nodeMaterialFolder:
		files := []model.Material{}
		for _, material := range m.visibleMaterials() {
			if material.CourseID == n.courseID && sectionLabel(material.Section) == n.section && material.Folder == n.mat.Folder {
				files = append(files, material)
			}
		}
		return files
	case nodeCourse:
		return m.courseFiles(n.courseID)
	}
	return nil
}

// materialNodeIDs lists the material IDs a row stands for: the file itself,
// every file in a week, folder, or course.
func (m *App) materialNodeIDs(n node) []string {
	return idsOf(m.materialNodeFiles(n))
}

// materialNodeSelected reports whether every material behind a row is marked,
// so group rows show the same mark state as their file rows.
func (m *App) materialNodeSelected(n node) bool {
	if n.kind == nodeFile {
		return m.selectedM[n.mat.ID]
	}
	return allSelected(m.selectedM, m.materialNodeIDs(n))
}

func materialKey(id string) string { return "material:" + id }

func (m *App) assignmentForMaterial(material model.Material) *model.Assignment {
	if material.AssignmentID != "" {
		return m.byID(material.AssignmentID)
	}
	if material.Kind != "attachment" || !strings.HasPrefix(material.ID, "M") {
		return nil
	}
	id := strings.TrimPrefix(strings.SplitN(material.ID, ":", 2)[0], "M")
	return m.byID("K" + id)
}

func (m *App) courseFiles(courseID int) []model.Material {
	if courseID == localCourseID {
		return m.unknownMaterials()
	}
	out := []model.Material{}
	for _, mt := range m.materials {
		if mt.CourseID == courseID {
			out = append(out, mt)
		}
	}
	return out
}

// download fetches the selected or focused materials. Sources that cannot
// transfer bytes (the demo dataset) keep the old flag-only behaviour.
func (m *App) download() tea.Cmd {
	if m.tab != TabMaterials {
		m.toast = "switch to Materials to download"
		m.toastTicks = 20
		return nil
	}
	if m.offline {
		m.toast = "offline · press s to retry"
		m.toastTicks = 30
		return nil
	}
	targets := m.materialTargets()
	if len(targets) == 0 {
		m.toast = "nothing new to download"
		m.toastTicks = 20
		return nil
	}

	_, canDownload := m.src.(source.Downloader)
	var cmds []tea.Cmd
	var flaggedIDs []string
	started, links, flagged, missing, local := 0, 0, 0, 0, 0
	for _, material := range targets {
		if material.Kind == "local" {
			local++
			continue
		}
		if material.Downloaded {
			continue
		}
		if material.Kind == "url" || material.Kind == "praktikum" || material.Kind == "book" || material.Kind == "page" {
			links++
			continue
		}
		if !canDownload {
			if target := m.materialByID(material.ID); target != nil {
				target.Downloaded = true
			}
			flaggedIDs = append(flaggedIDs, material.ID)
			flagged++
			continue
		}
		if material.URL == "" {
			missing++
			continue
		}
		if m.downloads[material.ID] {
			continue
		}
		m.downloads[material.ID] = true
		cmds = append(cmds, m.startDownload(material))
		started++
	}

	m.selectedM = map[string]bool{}
	switch {
	case started > 0:
		m.toast = fmt.Sprintf("downloading %d %s…", started, plural(started, "file", "files"))
	case local > 0:
		m.toast = "local file: press o to open"
	case links > 0:
		m.toast = "external link: press o to open in browser"
	case missing > 0:
		m.toast = "no download link on this row"
	case flagged > 0:
		m.toast = fmt.Sprintf("marked %d %s downloaded", flagged, plural(flagged, "file", "files"))
	default:
		m.toast = "nothing new to download"
	}
	m.toastTicks = 24
	if len(flaggedIDs) > 0 {
		m.updateState(func(s *store.State) {
			for _, id := range flaggedIDs {
				s.SetDownloaded(id, true)
			}
		})
	}
	return tea.Batch(cmds...)
}

const localCourseID = -1

// unknownMaterials turns files found on disk but absent from the source into
// synthetic, read-only rows for the `unknown` Materials filter.
func (m *App) unknownMaterials() []model.Material {
	out := make([]model.Material, 0, len(m.unknownFiles))
	for _, path := range m.unknownFiles {
		name := path
		if rel, err := filepath.Rel(m.cfg.LibraryDir, path); err == nil && !strings.HasPrefix(rel, "..") {
			name = rel
		}
		out = append(out, model.Material{
			ID: "local:" + path, CourseID: localCourseID, CourseCode: "local",
			Name: name, Section: "unknown", Kind: "local", Downloaded: true, URL: path,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (m *App) downloadedMaterials() int {
	n := 0
	for _, material := range m.materials {
		if material.Downloaded {
			n++
		}
	}
	return n
}
