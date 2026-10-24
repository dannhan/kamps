package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

func (m *App) taskGroupIDs(group node) []string {
	var ids []string
	for _, assignment := range m.visible() {
		include := false
		switch group.kind {
		case nodeTerm:
			course, ok := m.courseByID(assignment.CourseID)
			term := course.Term
			if term == "" {
				term = "other"
			}
			include = term == group.term || !ok && group.term == "other"
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
	byTerm := make(map[string][]int)
	for _, course := range m.courses {
		if len(byCourse[course.ID]) == 0 {
			continue
		}
		courses[course.ID] = course
		term := course.Term
		if term == "" {
			term = "other"
		}
		byTerm[term] = append(byTerm[term], course.ID)
	}
	for courseID := range byCourse {
		if _, ok := courses[courseID]; ok {
			continue
		}
		courses[courseID] = model.Course{ID: courseID, Short: fmt.Sprintf("course-%d", courseID), Term: "other"}
		byTerm["other"] = append(byTerm["other"], courseID)
	}

	terms := make([]string, 0, len(byTerm))
	for term := range byTerm {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	out := make([]node, 0, len(visible)*2)
	for _, term := range terms {
		termKey := "term:" + term
		out = append(out, node{kind: nodeTerm, term: term, key: termKey})
		if !m.taskIsOpen(termKey, true) {
			continue
		}
		courseIDs := byTerm[term]
		sort.Slice(courseIDs, func(i, j int) bool {
			left, right := courses[courseIDs[i]], courses[courseIDs[j]]
			leftName, rightName := left.Short, right.Short
			if leftName == "" {
				leftName = left.Code
			}
			if rightName == "" {
				rightName = right.Code
			}
			return leftName < rightName
		})
		for _, courseID := range courseIDs {
			courseKeyValue := courseKey(courseID)
			parents := []string{termKey}
			out = append(out, node{kind: nodeCourse, courseID: courseID, term: term, key: courseKeyValue, parents: parents})
			if !m.taskIsOpen(courseKeyValue, true) {
				continue
			}
			buckets := []struct {
				name string
				keep func(model.Assignment) bool
				open bool
			}{
				{"needs decision", func(a model.Assignment) bool { return a.Triage == model.Undecided }, true},
				{"tracked", func(a model.Assignment) bool { return a.Triage == model.Tracked }, true},
				{"closed", func(a model.Assignment) bool { return a.Triage == model.Missed || a.Triage == model.Ignored }, false},
			}
			for _, bucket := range buckets {
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
				bucketParents := append(append([]string(nil), parents...), courseKeyValue)
				out = append(out, node{kind: nodeBucket, courseID: courseID, term: term, bucket: bucket.name, key: key, parents: bucketParents})
				if !m.taskIsOpen(key, bucket.open) {
					continue
				}
				for _, assignment := range assignments {
					taskNodeKey := taskKey(assignment.ID)
					taskParents := append(append([]string(nil), bucketParents...), key)
					out = append(out, node{kind: nodeTask, courseID: courseID, term: term, bucket: bucket.name, key: taskNodeKey, parents: taskParents, task: assignment})
					if !m.taskIsOpen(taskNodeKey, false) {
						continue
					}
					for _, material := range m.assignmentMaterials(assignment) {
						materialParents := append(append([]string(nil), taskParents...), taskNodeKey)
						out = append(out, node{kind: nodeTaskMaterial, courseID: courseID, term: term, bucket: bucket.name, key: taskMaterialKey(material.ID), parents: materialParents, mat: material, task: assignment})
					}
				}
			}
		}
	}
	return out
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
