package tui

import (
	"fmt"
	"testing"
)

func TestBlockWindowScrolloff(t *testing.T) {
	blocks := make([][]string, 20)
	for i := range blocks {
		blocks[i] = []string{fmt.Sprint(i)}
	}
	start := 0
	for cursor := 0; cursor <= 10; cursor++ {
		_, start, _ = blockWindow(blocks, cursor, 9, start, 2)
	}
	if start != 4 {
		t.Fatalf("moving down: start = %d, want 4", start)
	}
	for cursor := 9; cursor >= 4; cursor-- {
		_, start, _ = blockWindow(blocks, cursor, 9, start, 2)
	}
	if start != 2 {
		t.Fatalf("moving up: start = %d, want 2", start)
	}
	_, start, _ = blockWindow(blocks, 9, 9, 0, 0)
	if start != 1 {
		t.Fatalf("zero scrolloff: start = %d, want 1", start)
	}
}

func TestBlockWindowWrappedAndSmall(t *testing.T) {
	blocks := [][]string{{"a"}, {"b", "b2", "b3"}, {"c"}, {"d", "d2"}, {"e"}, {"f"}}
	rows, start, end := blockWindow(blocks, 3, 6, 0, 2)
	if start != 2 || end != 6 || len(rows) != 5 {
		t.Fatalf("wrapped blocks: start=%d end=%d rows=%v", start, end, rows)
	}
	rows, start, end = blockWindow(blocks, 1, 3, 0, 5)
	if start != 1 || end != 2 || len(rows) != 3 {
		t.Fatalf("small viewport: start=%d end=%d rows=%v", start, end, rows)
	}
}

func TestBlockWindowAfterContentShrinksOrResizes(t *testing.T) {
	blocks := [][]string{{"a"}, {"b", "b2"}, {"c"}, {"d"}, {"e"}}
	rows, start, end := blockWindow(blocks, 3, 10, 3, 0)
	if start != 0 || end != 5 || len(rows) != 6 {
		t.Fatalf("after resize: start=%d end=%d rows=%v", start, end, rows)
	}
	rows, start, end = blockWindow(blocks[:4], 3, 4, 3, 0)
	if start != 1 || end != 4 || len(rows) != 4 {
		t.Fatalf("after shrink: start=%d end=%d rows=%v", start, end, rows)
	}
}

func TestCourseContextWindowRemainsStable(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.cfg.Scrolloff = 2
	nodes := []node{{kind: nodeCourse, key: "course", courseID: 1}}
	blocks := [][]string{{"heading"}}
	for i := 0; i < 12; i++ {
		nodes = append(nodes, node{parents: []string{"course"}, courseID: 1})
		blocks = append(blocks, []string{fmt.Sprint(i)})
	}
	for height := 4; height <= 9; height++ {
		for cursor := 1; cursor < len(nodes); cursor++ {
			for initial := 0; initial < len(nodes); initial++ {
				offset := initial
				rows, start, end := app.courseContextWindow(nodes, blocks, cursor, 40, height, &offset)
				rows2, start2, end2 := app.courseContextWindow(nodes, blocks, cursor, 40, height, &offset)
				if start != start2 || end != end2 || fmt.Sprint(rows) != fmt.Sprint(rows2) {
					t.Fatalf("height=%d cursor=%d initial=%d: %d:%d %v then %d:%d %v", height, cursor, initial, start, end, rows, start2, end2, rows2)
				}
			}
		}
	}
}

func TestCourseContextDoesNotHideTallFocusedBlock(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.cfg.Scrolloff = 2
	nodes := []node{{kind: nodeCourse, key: "course", courseID: 1}, {parents: []string{"course"}, courseID: 1}, {parents: []string{"course"}, courseID: 1}}
	blocks := [][]string{{"heading"}, {"other"}, {"one", "two", "three", "four"}}
	offset := 1
	rows, start, end := app.courseContextWindow(nodes, blocks, 2, 40, 4, &offset)
	if start != 2 || end != 3 || len(rows) != 4 || rows[0] != "one" {
		t.Fatalf("focused block not visible: start=%d end=%d rows=%v", start, end, rows)
	}
}
