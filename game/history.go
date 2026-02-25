//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
)

const historyKey = "rogueHistory"
const classWinsKey = "rogueClassWins"
const hintKey = "rogueHintSeen"
const maxHistoryRuns = 10

type RunRecord struct {
	Class      string
	Outcome    string // "Victory" or "Died"
	Floor      int
	Kills      int
	Gold       int
	Turns      int
	Difficulty int
	IsDaily    bool
}

func (r RunRecord) encode() string {
	daily := "0"
	if r.IsDaily {
		daily = "1"
	}
	return fmt.Sprintf("%s|%s|%d|%d|%d|%d|%d|%s",
		r.Class, r.Outcome, r.Floor, r.Kills, r.Gold, r.Turns, r.Difficulty, daily)
}

func decodeRun(s string) (RunRecord, bool) {
	parts := strings.Split(s, "|")
	if len(parts) < 6 {
		return RunRecord{}, false
	}
	floor, e1 := strconv.Atoi(parts[2])
	kills, e2 := strconv.Atoi(parts[3])
	gold, e3 := strconv.Atoi(parts[4])
	turns, e4 := strconv.Atoi(parts[5])
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return RunRecord{}, false
	}
	r := RunRecord{
		Class:   parts[0],
		Outcome: parts[1],
		Floor:   floor,
		Kills:   kills,
		Gold:    gold,
		Turns:   turns,
	}
	// Backward compat: fields 6+ may not exist
	if len(parts) >= 7 {
		r.Difficulty, _ = strconv.Atoi(parts[6])
	}
	if len(parts) >= 8 {
		r.IsDaily = parts[7] == "1"
	}
	return r, true
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

// ClassWins tracks victories per class name.
type ClassWins map[string]int

func loadClassWins() ClassWins {
	w := make(ClassWins)
	ls := js.Global().Get("localStorage")
	val := ls.Call("getItem", classWinsKey)
	if val.IsNull() || val.IsUndefined() {
		return w
	}
	s := val.String()
	if s == "" || s == "null" {
		return w
	}
	for _, pair := range strings.Split(s, "|") {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			n, err := strconv.Atoi(kv[1])
			if err == nil {
				w[kv[0]] = n
			}
		}
	}
	return w
}

func saveClassWins(wins ClassWins) {
	var parts []string
	for k, v := range wins {
		parts = append(parts, fmt.Sprintf("%s:%d", k, v))
	}
	ls := js.Global().Get("localStorage")
	ls.Call("setItem", classWinsKey, strings.Join(parts, "|"))
}

func (g *Game) recordRun(outcome string) {
	if g.ClassName == "" || g.Player == nil {
		return
	}
	stopAmbient()
	r := RunRecord{
		Class:      g.ClassName,
		Outcome:    outcome,
		Floor:      g.Floor,
		Kills:      g.Kills,
		Gold:       g.Player.Gold,
		Turns:      g.Turns,
		Difficulty: g.Difficulty,
		IsDaily:    g.IsDaily,
	}
	history := loadRunHistory()
	history = append([]RunRecord{r}, history...) // newest first
	if len(history) > maxHistoryRuns {
		history = history[:maxHistoryRuns]
	}
	saveRunHistory(history)
	g.RunHistory = history

	// Update class wins on victory
	if outcome == "Victory" {
		wins := loadClassWins()
		wins[g.ClassName]++
		saveClassWins(wins)
		g.ClassWins = wins
	}
}

// hintSeen returns true if the first-run control hint has been dismissed.
func hintSeen() bool {
	val := js.Global().Get("localStorage").Call("getItem", hintKey)
	return !val.IsNull() && !val.IsUndefined()
}

// markHintSeen saves the hint-seen flag so it never shows again.
func markHintSeen() {
	js.Global().Get("localStorage").Call("setItem", hintKey, "1")
}
