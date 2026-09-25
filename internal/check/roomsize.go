package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// checkHabitableRoomSize evaluates minimum floor area and minimum
// horizontal dimension (checked in the worst-case direction) for a room.
func checkHabitableRoomSize(section string, r rules.Rule, rm *room.Room) []Finding {
	var out []Finding

	label := "habitable room area"
	if r.IsUnverified() || r.MinAreaSqft == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinAreaSqft
		area := rm.FloorAreaSqft()
		ok := atLeast(area, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %.2f sqft min", fmtSqft(area), limit),
			area, limit, true, fmtSqft))
	}

	label = "habitable room minimum dimension"
	if r.IsUnverified() || r.MinHorizontalDimensionIn == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinHorizontalDimensionIn
		dim := rm.MinHorizontalDimensionIn()
		ok := atLeast(dim, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %s min", fmtIn(dim), fmtIn(limit)),
			dim, limit, true, fmtIn))
	}

	return out
}
