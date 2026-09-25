// Package county defines the public-data boundary for address-based site studies.
package county

import "context"

// Point is a longitude/latitude or projected coordinate pair, identified by CRS.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Feature is a public GIS geometry with its provenance. Coordinates retain the
// source service's CRS so consumers cannot mistake projected feet for degrees.
type Feature struct {
	Source string    `json:"source"`
	CRS    string    `json:"crs"`
	Rings  [][]Point `json:"rings,omitempty"`
	Point  *Point    `json:"point,omitempty"`
}

type Address struct {
	Label    string `json:"label"`
	ParcelID string `json:"parcel_id"`
	City     string `json:"city"`
	Location Point  `json:"location"`
}

// Site is the portable result returned by every county adapter.
type Site struct {
	Address           Address   `json:"address"`
	Parcel            Feature   `json:"parcel"`
	Footprints        []Feature `json:"footprints"`
	TerrainElevationM *float64  `json:"terrain_elevation_m,omitempty"`
	AerialURL         string    `json:"aerial_url"`
	Notes             []string  `json:"notes,omitempty"`
}

// Adapter is the contribution surface. Implementations must use publicly
// accessible sources, identify every source, and return an error rather than
// manufacturing a parcel or building outline.
type Adapter interface {
	County() string
	Lookup(context.Context, string) (Site, error)
}
