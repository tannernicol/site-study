package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// checkLightVentilation evaluates the glazing and openable-area percentages
// (of floor area) for a room, summed across all its windows.
func checkLightVentilation(section string, r rules.Rule, rm *room.Room) []Finding {
	var out []Finding
	floorArea := rm.FloorAreaSqft()

	label := "glazing area (% of floor area)"
	if r.IsUnverified() || r.MinGlazingPctOfFloorArea == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinGlazingPctOfFloorArea
		pct := pctOfFloorArea(rm.TotalGlazingAreaSqft(), floorArea)
		ok := atLeast(pct, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %.2f%% min", fmtPct(pct), limit),
			pct, limit, true, fmtPct))
	}

	label = "openable area (% of floor area)"
	if r.IsUnverified() || r.MinOpenablePctOfFloorArea == nil {
		out = append(out, skip(section, r, label, "unverified", "unverified rule"))
	} else {
		limit := *r.MinOpenablePctOfFloorArea
		pct := pctOfFloorArea(rm.TotalOpenableAreaSqft(), floorArea)
		ok := atLeast(pct, limit)
		out = append(out, dimFinding(section, r, label, passFail(ok),
			fmt.Sprintf("%s >= %.2f%% min", fmtPct(pct), limit),
			pct, limit, true, fmtPct))
	}

	return out
}

func pctOfFloorArea(areaSqft, floorAreaSqft float64) float64 {
	if floorAreaSqft == 0 {
		return 0
	}
	return areaSqft / floorAreaSqft * 100
}
