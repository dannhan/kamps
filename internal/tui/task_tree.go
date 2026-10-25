package tui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

// taskBuckets is the fixed order of state groups under every course row.
var taskBuckets = []struct {
	name string
	keep func(model.Assignment) bool
	open bool
}{
	{"needs decision", func(a model.Assignment) bool { return a.Triage == model.Undecided }, true},
	{"tracked", func(a model.Assignment) bool { return a.Triage == model.Tracked }, false},
	{"closed", func(a model.Assignment) bool { return a.Triage == model.Missed || a.Triage == model.Ignored }, false},
}

func (m *App) taskGroupIDs(group node) []string {
	var ids []string
	for _, assignment := range m.visible() {
		include := false
		switch group.kind {
		case nodeCourse:
			include = assignment.CourseID == group.courseID
		case nodeBucket:
			include = assignment.CourseID == group.courseID && (group.bucket == "needs decision" && assignment.Triage == model.Undecided || group.bucket == "tracked" && assignment.Triage == model.Tracked || group.bucket == "closed" && (assignment.Triage == model.Missed || assignment.Triage == model.Ignored))
		case nodeTask:
			include = assignment.ID == group.task.ID
		}
		if include {
			ids = append(ids, assignment.ID)
		}
	}
	return ids
}

// taskNodes flattens the Tasks tab: course ▸ state bucket ▸ task ▸ attached
// material. Courses are top-level rows; courses are ordered by term and then by
// their display name so a multi-semester set stays readable.
func (m *App) taskNodes() []node {
	visible := m.visible()
	byCourse := make(map[int][]model.Assignment)
	for _, assignment := range visible {
		byCourse[assignment.CourseID] = append(byCourse[assignment.CourseID], assignment)
	}
	if len(byCourse) == 0 {
		return nil
	}

	courses := make(map[int]model.Course, len(m.courses))
	for _, course := range m.courses {
		if len(byCourse[course.ID]) == 0 {
			continue
		}
		courses[course.ID] = course
	}
	for courseID := range byCourse {
		if _, ok := courses[courseID]; ok {
			continue
		}
		courses[courseID] = model.Course{ID: courseID, Short: "course-" + strconv.Itoa(courseID)}
	}

	courseIDs := make([]int, 0, len(courses))
	for courseID := range courses {
		courseIDs = append(courseIDs, courseID)
	}
	sort.Slice(courseIDs, func(i, j int) bool {
		left, right := courses[courseIDs[i]], courses[courseIDs[j]]
		if left.Term != right.Term {
			return left.Term < right.Term
		}
		return courseSortName(left) < courseSortName(right)
	})

	out := make([]node, 0, len(visible)*2)
	for _, courseID := range courseIDs {
		courseKeyValue := courseKey(courseID)
		out = append(out, node{kind: nodeCourse, courseID: courseID, key: courseKeyValue})
		if !m.taskIsOpen(courseKeyValue, true) {
			continue
		}
		for _, bucket := range taskBuckets {
			var assignments []model.Assignment
			for _, assignment := range byCourse[courseID] {
				if bucket.keep(assignment) {
					assignments = append(assignments, assignment)
				}
			}
			if len(assignments) == 0 {
				continue
			}
			key := bucketKey(courseID, bucket.name)
			out = append(out, node{kind: nodeBucket, courseID: courseID, bucket: bucket.name, key: key, parents: []string{courseKeyValue}})
			if !m.taskIsOpen(key, bucket.open) {
				continue
			}
			for _, assignment := range assignments {
				taskNodeKey := taskKey(assignment.ID)
				out = append(out, node{kind: nodeTask, courseID: courseID, bucket: bucket.name, key: taskNodeKey, parents: []string{courseKeyValue, key}, task: assignment})
				if !m.taskIsOpen(taskNodeKey, false) {
					continue
				}
				for _, material := range m.assignmentMaterials(assignment) {
					out = append(out, node{kind: nodeTaskMaterial, courseID: courseID, bucket: bucket.name, key: taskMaterialKey(material.ID), parents: []string{courseKeyValue, key, taskNodeKey}, mat: material, task: assignment})
				}
			}
		}
	}
	return out
}

func courseSortName(course model.Course) string {
	if course.Short != "" {
		return course.Short
	}
	return course.Code
}

func (m *App) assignmentMaterials(assignment model.Assignment) []model.Material {
	prefix := "M" + strings.TrimPrefix(assignment.ID, "K") + ":"
	var materials []model.Material
	for _, material := range m.materials {
		if strings.HasPrefix(material.ID, prefix) {
			materials = append(materials, material)
		}
	}
	sort.Slice(materials, func(i, j int) bool { return materials[i].Name < materials[j].Name })
	return materials
}

func (m *App) courseByID(id int) (model.Course, bool) {
	for _, c := range m.courses {
		if c.ID == id {
			return c, true
		}
	}
	return model.Course{}, false
}

func (m *App) courseChildren(courseID int) []model.Assignment {
	out := []model.Assignment{}
	for _, a := range m.visible() {
		if a.CourseID == courseID {
			out = append(out, a)
		}
	}
	return out
}
