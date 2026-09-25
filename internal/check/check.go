// Package check turns a rules.Rules + room.Project into a list of cited,
// confidence-tagged Findings. Every rule that structurally applies to some
// entity in the project produces at least one Finding — a rule is never
// silently dropped just because its numbers are missing or unverified.
package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// Status is the outcome of one finding.
type Status string

// The four outcomes a single measurement can have. Skipped means the rule
// could not be evaluated (unverified or missing limit), which is distinct
// from passing it.
const (
	Pass    Status = "PASS"
	Fail    Status = "FAIL"
	Flag    Status = "FLAG"
	Skipped Status = "SKIPPED"
)

// Finding is one printed line: a single measurement checked against a
// single rule, with its citation and confidence carried along so the
// output never separates a number from its source.
type Finding struct {
	Section    string `json:"section"` // room/stair name, or "Project"
	RuleID     string `json:"rule_id"`
	Label      string `json:"label"` // human label for this measurement
	Status     Status `json:"status"`
	Detail     string `json:"detail"`           // e.g. `81" >= 80" min`
	Actual     string `json:"actual,omitempty"` // e.g. `81"` — empty when there is no single measured value (SKIPPED, boolean triggers)
	Limit      string `json:"limit,omitempty"`  // e.g. `80" min`
	Margin     string `json:"margin,omitempty"` // signed distance from the limit, e.g. `+1"` or `-2"`
	Reason     string `json:"reason,omitempty"` // why SKIPPED
	Citation   string `json:"citation"`
	Confidence string `json:"confidence"`
}

// skip builds a SKIPPED finding for a rule whose numbers must never be
// trusted — either because the rule itself is unverified/null, or because
// the room spec has no data to check it against. Never silently pass.
//
// kind is the short reason shown in the status word, e.g. "SKIPPED
// (unverified)" or "SKIPPED (no data)"; detail is the longer explanation
// shown in the detail column.
func skip(section string, r rules.Rule, label, kind, detail string) Finding {
	return Finding{
		Section:    section,
		RuleID:     r.ID,
		Label:      label,
		Status:     Skipped,
		Detail:     detail,
		Reason:     kind,
		Citation:   r.Citation,
		Confidence: r.Confidence,
	}
}

// atLeast checks value against a required minimum. limit == nil means no
// verified number exists, so the caller must skip instead of calling this.
func atLeast(value, limit float64) bool { return value >= limit }

// atMost checks value against a required maximum.
func atMost(value, limit float64) bool { return value <= limit }

func passFail(ok bool) Status {
	if ok {
		return Pass
	}
	return Fail
}

// fmtIn formats an inch measurement, dropping a trailing .00 for whole
// numbers but keeping precision otherwise.
func fmtIn(v float64) string {
	return fmt.Sprintf("%s\"", fmtNum(v))
}

func fmtNum(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.2f", v)
}

func fmtSqft(v float64) string {
	return fmt.Sprintf("%.2f sqft", v)
}

func fmtPct(v float64) string {
	return fmt.Sprintf("%.2f%%", v)
}

// Run evaluates every rule against every matching entity in the project and
// returns the full, ordered set of findings.
func Run(rs *rules.Rules, proj *room.Project) []Finding {
	// One dispatcher per rule shape. Keeping the per-shape loops out of Run is
	// what keeps this readable as the rule set grows — the switch stays a table
	// of contents rather than the whole book.
	var findings []Finding
	for _, r := range rs.Checks {
		if fn, ok := checkers[shapeOf(r)]; ok {
			findings = append(findings, fn(r, proj)...)
		}
	}
	return findings
}

var checkers = map[shape]func(rules.Rule, *room.Project) []Finding{
	shapeCeilingHeight:     runCeilingHeight,
	shapeEgressOpening:     runEgressOpening,
	shapeWindowWell:        runWindowWell,
	shapeHabitableRoomSize: runHabitableRoomSize,
	shapeLightVentilation:  runLightVentilation,
	shapeAlarms:            runAlarms,
	shapeStairs:            runStairs,
	shapeInsulation:        runInsulation,
	shapeZoning:            runZoning,
}

func runCeilingHeight(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Rooms {
		rm := &proj.Rooms[i]
		facts := map[string]any{
			"space_type":        rm.SpaceType,
			"existing_basement": proj.ExistingBasement,
		}
		if r.Matches(facts) {
			out = append(out, checkCeilingHeight(rm.Name, r, rm)...)
		}
	}
	return out
}

func runEgressOpening(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Rooms {
		rm := &proj.Rooms[i]
		facts := map[string]any{"requires_eero": rm.RequiresEero}
		if r.Matches(facts) {
			for j := range rm.Windows {
				out = append(out, checkEgressOpening(rm.Name, r, &rm.Windows[j])...)
			}
		}
	}
	return out
}

func runWindowWell(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Rooms {
		rm := &proj.Rooms[i]
		for j := range rm.Windows {
			w := &rm.Windows[j]
			facts := map[string]any{
				"requires_eero":    rm.RequiresEero,
				"sill_below_grade": w.SillBelowGrade,
			}
			if r.Matches(facts) {
				out = append(out, checkWindowWell(rm.Name, r, w)...)
			}
		}
	}
	return out
}

func runHabitableRoomSize(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Rooms {
		rm := &proj.Rooms[i]
		facts := map[string]any{"space_type": rm.SpaceType}
		if r.Matches(facts) {
			out = append(out, checkHabitableRoomSize(rm.Name, r, rm)...)
		}
	}
	return out
}

func runLightVentilation(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Rooms {
		rm := &proj.Rooms[i]
		facts := map[string]any{"space_type": rm.SpaceType}
		if r.Matches(facts) {
			out = append(out, checkLightVentilation(rm.Name, r, rm)...)
		}
	}
	return out
}

func runAlarms(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Rooms {
		rm := &proj.Rooms[i]
		facts := map[string]any{"space_type": rm.SpaceType}
		if r.Matches(facts) {
			out = append(out, checkAlarms(rm.Name, r)...)
		}
	}
	return out
}

func runStairs(r rules.Rule, proj *room.Project) []Finding {
	var out []Finding
	for i := range proj.Stairs {
		st := &proj.Stairs[i]
		facts := map[string]any{"element": "stair"}
		if r.Matches(facts) {
			out = append(out, checkStairs(st.Name, r, st)...)
		}
	}
	return out
}

func runInsulation(r rules.Rule, proj *room.Project) []Finding {
	facts := map[string]any{"element": "envelope"}
	if r.Matches(facts) {
		return checkInsulation(r)
	}
	return nil
}

// runZoning is project-level like runInsulation, but does not gate on
// r.Matches: a zoning rule's applies_when also carries `zone` (which zone
// it's written for, e.g. "LR1 (M)") and room.Project has no zone fact to
// match it against -- checkZoning's own SKIP already says why, unconditionally,
// same as this rule shape always being real (Run's doc: every applicable
// rule produces at least one Finding).
func runZoning(r rules.Rule, proj *room.Project) []Finding {
	return checkZoning(r)
}

// Summary counts findings by status.
type Summary struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Flag    int `json:"flag"`
	Skipped int `json:"skipped"`
}

// Summarize tallies a finding list.
func Summarize(findings []Finding) Summary {
	var s Summary
	for _, f := range findings {
		switch f.Status {
		case Pass:
			s.Pass++
		case Fail:
			s.Fail++
		case Flag:
			s.Flag++
		case Skipped:
			s.Skipped++
		}
	}
	return s
}

// ExitCode returns the process exit code for a finding set: 1 if any FAIL,
// 0 otherwise (PASS/FLAG/SKIPPED are all advisory for exit purposes).
func ExitCode(findings []Finding) int {
	for _, f := range findings {
		if f.Status == Fail {
			return 1
		}
	}
	return 0
}
