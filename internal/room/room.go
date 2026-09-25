// Package room loads a room-spec YAML (tape measure, planner export, or a
// future IFC conversion) — the as-built facts that rules are checked
// against.
package room

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Window is one opening in a room.
type Window struct {
	Name string `yaml:"name"`

	NetClearWidthIn  float64 `yaml:"net_clear_width_in"`
	NetClearHeightIn float64 `yaml:"net_clear_height_in"`
	SillHeightIn     float64 `yaml:"sill_height_in"`
	SillBelowGrade   bool    `yaml:"sill_below_grade"`

	WellHorizontalIn float64 `yaml:"well_horizontal_in"`
	WellDepthIn      float64 `yaml:"well_depth_in"`

	GlazingAreaSqft  float64 `yaml:"glazing_area_sqft"`
	OpenableAreaSqft float64 `yaml:"openable_area_sqft"`
}

// NetClearAreaSqft computes the egress net clear opening area from the
// measured width and height. This is computed, never trusted from a stored
// field, so a stale/typo'd area value in the source YAML can't slip through.
func (w Window) NetClearAreaSqft() float64 {
	return w.NetClearWidthIn * w.NetClearHeightIn / 144
}

// WellAreaSqft computes the window well footprint from its measured
// horizontal dimension and depth.
func (w Window) WellAreaSqft() float64 {
	return w.WellHorizontalIn * w.WellDepthIn / 144
}

// Room is one space in the project.
type Room struct {
	Name      string  `yaml:"name"`
	SpaceType string  `yaml:"space_type"`
	LengthIn  float64 `yaml:"length_in"`
	WidthIn   float64 `yaml:"width_in"`

	ClearHeightIn float64 `yaml:"clear_height_in"`

	// LowestObstructionIn is a pointer: a room with no beam/duct crossing
	// it simply omits this field, which must not be treated as 0".
	LowestObstructionIn *float64 `yaml:"lowest_obstruction_in"`

	RequiresEero bool     `yaml:"requires_eero"`
	Windows      []Window `yaml:"windows"`
}

// FloorAreaSqft computes the room's floor area from its measured length and
// width.
func (r Room) FloorAreaSqft() float64 {
	return r.LengthIn * r.WidthIn / 144
}

// MinHorizontalDimensionIn returns the smaller of length/width — code checks
// the worst-case direction, not the average.
func (r Room) MinHorizontalDimensionIn() float64 {
	if r.LengthIn < r.WidthIn {
		return r.LengthIn
	}
	return r.WidthIn
}

// TotalGlazingAreaSqft sums glazing area across all windows in the room.
func (r Room) TotalGlazingAreaSqft() float64 {
	var total float64
	for _, w := range r.Windows {
		total += w.GlazingAreaSqft
	}
	return total
}

// TotalOpenableAreaSqft sums openable area across all windows in the room.
func (r Room) TotalOpenableAreaSqft() float64 {
	var total float64
	for _, w := range r.Windows {
		total += w.OpenableAreaSqft
	}
	return total
}

// Stair is one stairway element.
type Stair struct {
	Name string `yaml:"name"`

	HeadroomIn float64 `yaml:"headroom_in"`
	RiserIn    float64 `yaml:"riser_in"`
	TreadIn    float64 `yaml:"tread_in"`
	WidthIn    float64 `yaml:"width_in"`

	// Existing means the stair is original to the building: a geometry
	// violation here is a FLAG (possibly grandfathered), never a FAIL.
	Existing bool `yaml:"existing"`
}

// Project is the top-level room-spec document.
type Project struct {
	Project    string `yaml:"project"`
	Source     string `yaml:"source"`
	MeasuredOn string `yaml:"measured_on"`

	// ExistingBasement is available to local rule sets that distinguish existing
	// construction from new work. It is a project-level fact, not a per-room one.
	ExistingBasement bool `yaml:"existing_basement"`

	Rooms  []Room  `yaml:"rooms"`
	Stairs []Stair `yaml:"stairs"`
}

// Load reads and parses a room-spec YAML file.
func Load(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read room file %s: %w", path, err)
	}
	var p Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse room file %s: %w", path, err)
	}
	return &p, nil
}
