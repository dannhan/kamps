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
			if m.matchesMaterialSearch(material) && (!m.courseScoped || material.CourseID == m.courseID) {
				out = append(out, material)
			}
		}
		return out
	}
	items := m.service.Materials(core.Query{Filter: m.filter, Search: m.search, CourseID: m.courseID})
	out := make([]model.Material, 0, len(items))
	for _, item := range items {
		if !m.courseScoped || item.CourseID == m.courseID {
			out = append(out, item)
		}
	}
	return out
}

// matNodes presents a collapsed week overview for All courses, or the selected
// course's expanded library. Both use the same files and action targets.
func (m *App) matNodes() []node {
	bySection := map[string][]model.Material{}
	for _, material := range m.visibleMaterials() {
		section := sectionLabel(material.Section)
		bySection[section] = append(bySection[section], material)
	}
	sections := make([]string, 0, len(bySection))
	for section := range bySection {
		sections = append(sections, section)
	}
	sort.Slice(sections, func(i, j int) bool {
		_, leftIsWeek := core.SectionNumber(sections[i])
		_, rightIsWeek := core.SectionNumber(sections[j])
		if leftIsWeek != rightIsWeek {
			return leftIsWeek
		}
		if !leftIsWeek || m.materialsOldestFirst {
			return sectionLess(sections[i], sections[j])
		}
		return sectionLess(sections[j], sections[i])
	})
	var out []node
	for sectionIndex, section := range sections {
		files := bySection[section]
		if m.courseScoped {
			sectionKey := materialSectionKey(m.courseID, section)
			if sectionIndex > 0 {
				out = append(out, node{kind: nodeSpacer, key: "gap:" + sectionKey})
			}
			out = append(out, node{kind: nodeMaterialSection, courseID: m.courseID, section: section, key: sectionKey})
			if m.isOpen(sectionKey) {
				out = append(out, m.materialFileNodes(files, section, []string{sectionKey})...)
			}
			continue
		}
		weekKey := materialWeekKey(section)
		out = append(out, node{kind: nodeMaterialWeek, section: section, key: weekKey})
		if !m.isOpen(weekKey) {
			continue
		}
		byCourse := map[int][]model.Material{}
		for _, file := range files {
			byCourse[file.CourseID] = append(byCourse[file.CourseID], file)
		}
		courseIDs := make([]int, 0, len(byCourse))
		for courseID := range byCourse {
			courseIDs = append(courseIDs, courseID)
		}
		sort.Slice(courseIDs, func(i, j int) bool {
			left, right := m.fullCourseName(courseIDs[i]), m.fullCourseName(courseIDs[j])
			if left == right {
				return courseIDs[i] < courseIDs[j]
			}
			return left < right
		})
		for _, courseID := range courseIDs {
			key := "mwc:" + materialSectionKey(courseID, section)
			out = append(out, node{kind: nodeCourse, courseID: courseID, section: section, key: key, parents: []string{weekKey}})
			if m.isOpen(key) {
				out = append(out, m.materialFileNodes(byCourse[courseID], section, []string{weekKey, key})...)
			}
		}
		if sectionIndex < len(sections)-1 {
			out = append(out, node{kind: nodeSpacer, key: "gap:" + weekKey})
		}
	}
	return out
}

// materialFileNodes retains folder ownership and stable file keys in both views.
func (m *App) materialFileNodes(files []model.Material, section string, parents []string) []node {
	byFolder := map[string][]model.Material{}
	var folders []string
	var direct []model.Material
	for _, file := range files {
		if file.Folder == "" {
			direct = append(direct, file)
			continue
		}
		if _, exists := byFolder[file.Folder]; !exists {
			folders = append(folders, file.Folder)
		}
		byFolder[file.Folder] = append(byFolder[file.Folder], file)
	}
	var out []node
	appendFiles := func(items []model.Material, fileParents []string) {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Name == items[j].Name {
				return items[i].ID < items[j].ID
			}
			return items[i].Name < items[j].Name
		})
		for _, file := range items {
			out = append(out, node{kind: nodeFile, courseID: file.CourseID, mat: file, key: materialKey(file.ID), parents: fileParents})
		}
	}
	sort.Strings(folders)
	for _, folder := range folders {
		courseID := byFolder[folder][0].CourseID
		key := materialFolderKey(courseID, section, folder)
		if !m.courseScoped {
			key = "overview:" + key
		}
		out = append(out, node{kind: nodeMaterialFolder, courseID: courseID, section: section, key: key, parents: parents, mat: model.Material{Folder: folder}})
		if m.isOpen(key) {
			fileParents := append(append([]string(nil), parents...), key)
			appendFiles(byFolder[folder], fileParents)
		}
	}
	appendFiles(direct, parents)
	return out
}

func (m *App) materialSortLabel() string {
	if m.materialsOldestFirst {
		return "Oldest week first"
	}
	return "Newest week first"
}

func (m *App) toggleMaterialSort() {
	if m.tab != TabMaterials {
		return
	}
	nodes := m.matNodes()
	key := ""
	if m.matCursor >= 0 && m.matCursor < len(nodes) {
		key = nodes[m.matCursor].key
	}
	m.materialsOldestFirst = !m.materialsOldestFirst
	for index, n := range m.matNodes() {
		if n.key == key {
			m.matCursor = index
			break
		}
	}
	m.visual = false
	m.detailScroll = 0
	m.clampCursor()
}

// materialNodeFiles lists the materials a row stands for: the file itself,
// every visible file in a week or folder, or every file in a course.
func (m *App) materialNodeFiles(n node) []model.Material {
	switch n.kind {
	case nodeFile:
		return []model.Material{n.mat}
	case nodeMaterialWeek, nodeMaterialSection, nodeMaterialFolder, nodeCourse:
		var files []model.Material
		for _, material := range m.visibleMaterials() {
			if n.kind != nodeMaterialWeek && material.CourseID != n.courseID {
				continue
			}
			if n.section != "" && sectionLabel(material.Section) != n.section {
				continue
			}
			if n.kind == nodeMaterialFolder && material.Folder != n.mat.Folder {
				continue
			}
			files = append(files, material)
		}
		return files
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
		delete(m.downloadErrors, material.ID)
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
		m.service.ApplyDownloaded(flaggedIDs)
		m.service.ApplyState(func(s *store.State) {
			for _, id := range flaggedIDs {
				s.SetDownloaded(id, true)
			}
		})
		m.materials = m.service.Snapshot().Materials
		m.state = m.service.State()
		m.statusErr = ""
		for _, id := range flaggedIDs {
			m.pendingDownloads[id] = store.DownloadRecord{}
		}
		cmds = append(cmds, m.queueStatePersist())
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
