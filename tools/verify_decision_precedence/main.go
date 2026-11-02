package main

import (
	"fmt"
	"os"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/store"
)

var triages = []model.Triage{model.Undecided, model.Tracked, model.Missed, model.Ignored}

func main() {
	checked := 0
	for _, userDecision := range triages {
		for _, automaticDecision := range triages {
			check("automatic then user", userDecision, automaticDecision, true)
			check("user then automatic", userDecision, automaticDecision, false)
			checked += 2
		}
	}
	fmt.Printf("PASS: checked %d decision precedence cases against internal/store.State\n", checked)
}

func check(order string, userDecision, automaticDecision model.Triage, autoFirst bool) {
	state := store.New()
	if autoFirst {
		state.SetAutoDecision("K", automaticDecision)
	}
	state.SetDecision("K", userDecision)
	if !autoFirst {
		state.SetAutoDecision("K", automaticDecision)
	}

	assignments := []model.Assignment{{ID: "K", Triage: model.Undecided}}
	state.Apply(assignments, nil)
	if got := assignments[0].Triage; got != userDecision {
		fmt.Fprintf(os.Stderr, "FAIL: %s, user=%s, automatic=%s: got %s, want user decision %s\n", order, userDecision, automaticDecision, got, userDecision)
		os.Exit(1)
	}
}
