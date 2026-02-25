//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
)

const historyKey = "rogueHistory"
const maxHistoryRuns = 10

type RunRecord struct {
	Class   string
	Outcome string // "Victory" or "Died"
	Floor   int
	Kills   int
	Gold    int
	Turns   int
}

func (r RunRecord) encode() string {
	return fmt.Sprintf("%s|%s|%d|%d|%d|%d",
		r.Class, r.Outcome, r.Floor, r.Kills, r.Gold, r.Turns)
}

func decodeRun(s string) (RunRecord, bool) {
	parts := strings.Split(s, "|")
	if len(parts) != 6 {
		return RunRecord{}, false
	}
	floor, e1 := strconv.Atoi(parts[2])
	kills, e2 := strconv.Atoi(parts[3])
	gold, e3 := strconv.Atoi(parts[4])
	turns, e4 := strconv.Atoi(parts[5])
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return RunRecord{}, false
	}
	return RunRecord{
		Class:   parts[0],
		Outcome: parts[1],
		Floor:   floor,
		Kills:   kills,
		Gold:    gold,
		Turns:   turns,
	}, true
}

func loadRunHistory() []RunRecord {
	ls := js.Global().Get("localStorage")
	val := ls.Call("getItem", historyKey)
	if val.IsNull() || val.IsUndefined() {
		return nil
	}
	s := val.String()
	if s == "" || s == "null" {
		return nil
	}
	var runs []RunRecord
	for _, part := range strings.Split(s, ";") {
		if r, ok := decodeRun(part); ok {
			runs = append(runs, r)
		}
	}
	return runs
}

func saveRunHistory(runs []RunRecord) {
	parts := make([]string, len(runs))
	for i, r := range runs {
		parts[i] = r.encode()
	}
	ls := js.Global().Get("localStorage")
	ls.Call("setItem", historyKey, strings.Join(parts, ";"))
}

func (g *Game) recordRun(outcome string) {
	if g.ClassName == "" || g.Player == nil {
		return
	}
	r := RunRecord{
		Class:   g.ClassName,
		Outcome: outcome,
		Floor:   g.Floor,
		Kills:   g.Kills,
		Gold:    g.Player.Gold,
		Turns:   g.Turns,
	}
	history := loadRunHistory()
	history = append([]RunRecord{r}, history...) // newest first
	if len(history) > maxHistoryRuns {
		history = history[:maxHistoryRuns]
	}
	saveRunHistory(history)
	g.RunHistory = history
}
