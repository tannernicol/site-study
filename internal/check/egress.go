package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// checkEgressOpening evaluates the four egress-opening minimums against one
// window. The net clear area is always computed from the measured width and
// height — never read from a stored area field — so a stale area value
// can't slip past the check.
func checkEgressOpening(section string, r rules.Rule, w *room.Window) []Finding {
	label := fmt.Sprintf("%s — egress net clear width", w.Name)
	var out []Finding

	if r.IsUnverified() || r.MinNetClearWidthIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinNetClearWidthIn
		ok := atLeast(w.NetClearWidthIn, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %s min", fmtIn(w.NetClearWidthIn), fmtIn(limit)),
			w.NetClearWidthIn, limit, true, fmtIn))
	}

	label = fmt.Sprintf("%s — egress net clear height", w.Name)
	if r.IsUnverified() || r.MinNetClearHeightIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinNetClearHeightIn
		ok := atLeast(w.NetClearHeightIn, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %s min", fmtIn(w.NetClearHeightIn), fmtIn(limit)),
			w.NetClearHeightIn, limit, true, fmtIn))
	}

	label = fmt.Sprintf("%s — egress net clear area", w.Name)
	if r.IsUnverified() || r.MinNetClearAreaSqft == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinNetClearAreaSqft
		area := w.NetClearAreaSqft()
		ok := atLeast(area, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %.2f sqft min", fmtSqft(area), limit),
			area, limit, true, fmtSqft,
			"min_net_clear_area_sqft"))
	}

	label = fmt.Sprintf("%s — egress sill height", w.Name)
	if r.IsUnverified() || r.MaxSillHeightIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MaxSillHeightIn
		ok := atMost(w.SillHeightIn, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s <= %s max", fmtIn(w.SillHeightIn), fmtIn(limit)),
			w.SillHeightIn, limit, false, fmtIn))
	}

	return out
}

// finding is a small constructor shared by the checkers below to cut down
// on repeated field assignment.
// The optional field argument names the YAML limit key this finding was
// decided by, so a per-field confidence override in the rules file is
// reported instead of the rule-level one.
func finding(section string, r rules.Rule, label string, status Status, detail string, field ...string) Finding {
	confidence := r.Confidence
	if len(field) > 0 {
		confidence = r.ConfidenceFor(field[0])
	}
	return Finding{
		Section:    section,
		RuleID:     r.ID,
		Label:      label,
		Status:     status,
		Detail:     detail,
		Citation:   r.Citation,
		Confidence: confidence,
	}
}

// dimFinding wraps finding for a numeric measurement checked against a
// numeric limit, filling Actual/Limit/Margin from the same values and
// comparison direction used to build detail — so the report's structured
// columns and the prose Detail sentence can never drift apart. atLeastCmp
// is true when limit is a minimum (value >= limit), false when it's a
// maximum.
func dimFinding(section string, r rules.Rule, label string, status Status, detail string, actual, limit float64, atLeastCmp bool, format func(float64) string, field ...string) Finding {
	f := finding(section, r, label, status, detail, field...)
	bound := "max"
	if atLeastCmp {
		bound = "min"
	}
	f.Actual = format(actual)
	f.Limit = format(limit) + " " + bound
	f.Margin = marginStr(actual, limit, atLeastCmp, format)
	return f
}

// marginStr formats the signed distance between actual and limit — how much
// room there is, not just which side of the line — using the same
// comparison direction as the check itself.
func marginStr(actual, limit float64, atLeastCmp bool, format func(float64) string) string {
	m := actual - limit
	if !atLeastCmp {
		m = limit - actual
	}
	if m >= 0 {
		return "+" + format(m)
	}
	return format(m)
}
