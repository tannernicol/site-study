# County adapters

`county.Adapter` is the boundary between the generic engine and public county
data:

```go
type Adapter interface {
	County() string
	Lookup(context.Context, string) (Site, error)
}
```

An adapter must resolve an address to exactly one public parcel, return the
parcel polygon and any public building-footprint polygons with their source
CRS, and return a terrain elevation and aerial-image URL when the public
service provides them. It must name every source and return an error for an
ambiguous address; it must never invent geometry.

To add a county, implement the interface in `internal/county`, unit-test its
HTTP requests against a local `httptest.Server`, then register it in the web
and MCP entry points. Keep public-service URLs and field mappings in the
adapter, not in callers.

King County's free public sources are its [GIS Center](https://kingcounty.gov/en/dept/assessor/property-tax-department/gis-maps), its ArcGIS REST services for address points and parcels, and the King County open-data catalog. The shipped adapter also uses the USGS 3DEP elevation service and Esri World Imagery. Verify endpoints and terms before production use.
