package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// checkWindowWell evaluates the window-well minimums for one below-grade
// egress window. The well area is always computed from the measured
// horizontal dimension and depth, never trusted from a stored field.
func checkWindowWell(section string, r rules.Rule, w *room.Window) []Finding {
	var out []Finding

	label := fmt.Sprintf("%s — well horizontal dimension", w.Name)
	if r.IsUnverified() || r.MinHorizontalDimensionIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinHorizontalDimensionIn
		ok := atLeast(w.WellHorizontalIn, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %s min", fmtIn(w.WellHorizontalIn), fmtIn(limit)),
			w.WellHorizontalIn, limit, true, fmtIn))
	}

	label = fmt.Sprintf("%s — well area", w.Name)
	if r.IsUnverified() || r.MinAreaSqft == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinAreaSqft
		area := w.WellAreaSqft()
		ok := atLeast(area, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %.2f sqft min", fmtSqft(area), limit),
			area, limit, true, fmtSqft,
			"min_area_sqft"))
	}

	// Ladder requirement is conditional, not a plain min/max: the room
	// spec has no field recording whether a ladder is actually installed,
	// so this can only ever tell the reader whether one is REQUIRED, never
	// confirm compliance. A triggered requirement is flagged for the
	// reader to verify in the field, never passed or failed outright.
	label = fmt.Sprintf("%s — well ladder", w.Name)
	if r.IsUnverified() || r.LadderRequiredIfDepthOverIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		threshold := *r.LadderRequiredIfDepthOverIn
		if w.WellDepthIn > threshold {
			out = append(out, finding(section, r, label, Flag,
				fmt.Sprintf("depth %s > %s — ladder required, verify installed", fmtIn(w.WellDepthIn), fmtIn(threshold)),
				"ladder_required_if_depth_over_in"))
		} else {
			out = append(out, finding(section, r, label, Pass,
				fmt.Sprintf("depth %s <= %s — ladder not required", fmtIn(w.WellDepthIn), fmtIn(threshold)),
				"ladder_required_if_depth_over_in"))
		}
	}

	return out
}
