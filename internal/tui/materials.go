package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

// visibleMaterials returns the Materials-tab files after the active filter and
// search. The tree, counters and cursor all read this one list.
func (m *App) visibleMaterials() []model.Material {
	out := make([]model.Material, 0, len(m.materials))
	for _, material := range m.materials {
		if !m.matchesMaterialFilter(material) || !m.matchesMaterialSearch(material) {
			continue
		}
		out = append(out, material)
	}
	return out
}

func (m *App) matchesMaterialFilter(material model.Material) bool {
	switch m.filter {
	case "new":
		return !material.Downloaded
	case "downloaded":
		return material.Downloaded
	case "link":
		return material.Kind == "url"
	}
	return true
}

// matNodes is the Materials pane: course ▸ state bucket ▸ file. Weeks keep the
// source's own label; an empty section shows as "general".
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
		buckets := []struct {
			label string
			keep  func(model.Material) bool
		}{{"downloaded", func(mt model.Material) bool { return mt.Downloaded }}, {"new", func(mt model.Material) bool { return !mt.Downloaded }}}
		for _, bucket := range buckets {
			var files []model.Material
			for _, material := range byCourse[courseID] {
				if bucket.keep(material) {
					files = append(files, material)
				}
			}
			if len(files) == 0 {
				continue
			}
			key := materialBucketKey(courseID, bucket.label)
			parents := []string{course.key}
			out = append(out, node{kind: nodeMaterialBucket, courseID: courseID, bucket: bucket.label, key: key, parents: parents})
			if !m.isOpen(key) {
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
			for _, file := range files {
				label := sectionLabel(file.Section)
				if _, ok := bySection[label]; !ok {
					sections = append(sections, label)
				}
				bySection[label] = append(bySection[label], file)
			}
			sort.Strings(sections)
			for _, section := range sections {
				sectionFiles := bySection[section]
				sectionKey := materialSectionKey(courseID, bucket.label, section)
				out = append(out, node{kind: nodeMaterialSection, courseID: courseID, bucket: bucket.label, section: section, key: sectionKey, parents: []string{course.key, key}})
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
					folderKey := materialFolderKey(courseID, bucket.label, section, folder)
					out = append(out, node{kind: nodeMaterialFolder, courseID: courseID, bucket: bucket.label, section: section, key: folderKey, parents: []string{course.key, key, sectionKey}, mat: model.Material{Folder: folder}})
					if !m.isOpen(folderKey) {
						continue
					}
					for _, file := range folderFiles {
						emitFile(file, []string{course.key, key, sectionKey, folderKey})
					}
				}
				sort.Slice(direct, func(i, j int) bool { return direct[i].Name < direct[j].Name })
				for _, file := range direct {
					emitFile(file, []string{course.key, key, sectionKey})
				}
			}
		}
	}
	return out
}

func materialBucketKey(courseID int, bucket string) string {
	return fmt.Sprintf("mb:%d:%s", courseID, bucket)
}

// materialNodeFiles lists the materials a row stands for: the file itself,
// every file in a state bucket, every file in a week, or every file in a
// course.
func (m *App) materialNodeFiles(n node) []model.Material {
	switch n.kind {
	case nodeFile:
		return []model.Material{n.mat}
	case nodeMaterialBucket:
		files := []model.Material{}
		for _, material := range m.courseFiles(n.courseID) {
			if n.bucket == "downloaded" && material.Downloaded || n.bucket == "new" && !material.Downloaded {
				files = append(files, material)
			}
		}
		return files
	case nodeMaterialSection:
		files := []model.Material{}
		for _, material := range m.courseFiles(n.courseID) {
			if !materialBucketMatch(n.bucket, material) {
				continue
			}
			if sectionLabel(material.Section) == n.section {
				files = append(files, material)
			}
		}
		return files
	case nodeMaterialFolder:
		files := []model.Material{}
		for _, material := range m.courseFiles(n.courseID) {
			if !materialBucketMatch(n.bucket, material) {
				continue
			}
			if sectionLabel(material.Section) == n.section && material.Folder == n.mat.Folder {
				files = append(files, material)
			}
		}
		return files
	case nodeCourse:
		return m.courseFiles(n.courseID)
	}
	return nil
}

func materialBucketMatch(bucket string, material model.Material) bool {
	return bucket == "downloaded" && material.Downloaded || bucket == "new" && !material.Downloaded
}

// materialNodeIDs lists the material IDs a row stands for: the file itself,
// every file in a state bucket, or every file in a course/week.
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
	targets := m.materialTargets()
	if len(targets) == 0 {
		m.toast = "nothing new to download"
		m.toastTicks = 20
		return nil
	}

	_, canDownload := m.src.(source.Downloader)
	var cmds []tea.Cmd
	var flaggedIDs []string
	started, links, flagged, missing := 0, 0, 0, 0
	for _, material := range targets {
		if material.Downloaded {
			continue
		}
		if material.Kind == "url" || material.Kind == "praktikum" || material.Kind == "book" {
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

func (m *App) downloadedMaterials() int {
	n := 0
	for _, material := range m.materials {
		if material.Downloaded {
			n++
		}
	}
	return n
}
