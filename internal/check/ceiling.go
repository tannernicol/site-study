package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// obstructionMarginIn is a field-verification judgment threshold, not a
// code number: when the measured obstruction clearance is within this many
// inches of the rule's minimum, the finding is flagged for a re-measure
// rather than passed silently, because a tape-measure error of that size
// could flip the result. It is never used as, or in place of, a code limit.
const obstructionMarginIn = 1.0

// checkCeilingHeight evaluates the "min clear height" and, if the rule
// defines one, the "min height under an obstruction" sub-checks for a room.
func checkCeilingHeight(section string, r rules.Rule, rm *room.Room) []Finding {
	var out []Finding

	const label = "ceiling height"
	if r.IsUnverified() || r.MinClearHeightIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinClearHeightIn
		ok := atLeast(rm.ClearHeightIn, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %s min", fmtIn(rm.ClearHeightIn), fmtIn(limit)),
			rm.ClearHeightIn, limit, true, fmtIn))
	}

	const obstructionLabel = "obstruction clearance"
	if r.MinUnderObstructionIn != nil && rm.LowestObstructionIn != nil {
		if r.IsUnverified() {
			out = append(out, skip(section, r, obstructionLabel, "unverified", "unverified rule"))
		} else {
			limit := *r.MinUnderObstructionIn
			obstruction := *rm.LowestObstructionIn
			if obstruction < limit {
				out = append(out, dimFinding(section, r, obstructionLabel, Fail,
					fmt.Sprintf("%s < %s min", fmtIn(obstruction), fmtIn(limit)),
					obstruction, limit, true, fmtIn))
			} else {
				margin := obstruction - limit
				status := Pass
				detail := fmt.Sprintf("%s >= %s min (%s margin)", fmtIn(obstruction), fmtIn(limit), fmtIn(margin))
				if margin <= obstructionMarginIn {
					status = Flag
					detail = fmt.Sprintf("%s — within %s of the %s min, verify in the field", fmtIn(obstruction), fmtIn(margin), fmtIn(limit))
				}
				out = append(out, dimFinding(section, r, obstructionLabel, status, detail,
					obstruction, limit, true, fmtIn))
			}
		}
	}

	return out
}
