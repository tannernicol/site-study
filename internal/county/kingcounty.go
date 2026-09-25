package county

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	addressURL   = "https://gismaps.kingcounty.gov/arcgis/rest/services/Address/KingCo_AddressPoints/MapServer/0/query"
	parcelURL    = "https://gismaps.kingcounty.gov/arcgis/rest/services/Property/KingCo_Parcels/MapServer/0/query"
	footprintURL = "https://services.arcgis.com/ZOyb2t4B0UYuYNYH/arcgis/rest/services/Building_Outlines_2023/FeatureServer/0/query"
	elevationURL = "https://elevation.nationalmap.gov/arcgis/rest/services/3DEPElevation/ImageServer/identify"
	aerialURL    = "https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/export"
)

type KingCounty struct{ Client *http.Client }

func (KingCounty) County() string { return "King County, WA" }
func (k KingCounty) client() *http.Client {
	if k.Client != nil {
		return k.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func query(ctx context.Context, client *http.Client, endpoint string, values url.Values, out any) error {
	values.Set("f", "json")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("public GIS returned %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out); err != nil {
		return err
	}
	return nil
}

func (k KingCounty) Lookup(ctx context.Context, address string) (Site, error) {
	address = strings.TrimSpace(address)
	if len(address) < 3 {
		return Site{}, fmt.Errorf("address must be at least 3 characters")
	}
	var addresses struct {
		Features []struct {
			Attributes struct {
				Address string `json:"ADDR_FULL"`
				PIN     string `json:"PIN"`
				City    string `json:"CTYNAME"`
			}
			Geometry struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			}
		} `json:"features"`
		Error any `json:"error"`
	}
	where := "UPPER(ADDR_FULL)='" + strings.ReplaceAll(strings.ToUpper(strings.Split(address, ",")[0]), "'", "''") + "'"
	if err := query(ctx, k.client(), addressURL, url.Values{"where": {where}, "outFields": {"ADDR_FULL,PIN,CTYNAME"}, "returnGeometry": {"true"}, "outSR": {"4326"}}, &addresses); err != nil {
		return Site{}, fmt.Errorf("King County address lookup: %w", err)
	}
	if len(addresses.Features) != 1 || addresses.Features[0].Attributes.PIN == "" {
		return Site{}, fmt.Errorf("address must resolve to exactly one King County parcel")
	}
	a := addresses.Features[0]
	site := Site{Address: Address{Label: a.Attributes.Address, ParcelID: a.Attributes.PIN, City: a.Attributes.City, Location: Point{a.Geometry.X, a.Geometry.Y}}, AerialURL: aerialLink(a.Geometry.X, a.Geometry.Y)}
	if err := k.parcel(ctx, a.Attributes.PIN, &site); err != nil {
		return Site{}, err
	}
	_ = k.footprints(ctx, &site)
	_ = k.elevation(ctx, &site)
	if len(site.Footprints) == 0 {
		site.Notes = append(site.Notes, "No public building outline was returned; this is not evidence that no building exists.")
	}
	return site, nil
}

func (k KingCounty) parcel(ctx context.Context, pin string, site *Site) error {
	var response struct {
		Features []struct {
			Geometry struct {
				Rings [][][]float64 `json:"rings"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := query(ctx, k.client(), parcelURL, url.Values{"where": {"PIN='" + pin + "'"}, "outFields": {"PIN"}, "returnGeometry": {"true"}, "outSR": {"2926"}}, &response); err != nil {
		return fmt.Errorf("King County parcel lookup: %w", err)
	}
	if len(response.Features) != 1 || len(response.Features[0].Geometry.Rings) == 0 {
		return fmt.Errorf("no unique parcel geometry for PIN %s", pin)
	}
	site.Parcel = Feature{Source: "King County Parcels", CRS: "EPSG:2926", Rings: rings(response.Features[0].Geometry.Rings)}
	return nil
}
func (k KingCounty) footprints(ctx context.Context, site *Site) error {
	geometry, _ := json.Marshal(map[string]any{"x": site.Address.Location.X, "y": site.Address.Location.Y, "spatialReference": map[string]int{"wkid": 4326}})
	var response struct {
		Features []struct {
			Geometry struct {
				Rings [][][]float64 `json:"rings"`
			} `json:"geometry"`
		} `json:"features"`
	}
	err := query(ctx, k.client(), footprintURL, url.Values{"geometry": {string(geometry)}, "geometryType": {"esriGeometryPoint"}, "inSR": {"4326"}, "spatialRel": {"esriSpatialRelIntersects"}, "outFields": {"OBJECTID"}, "returnGeometry": {"true"}, "outSR": {"2926"}}, &response)
	if err != nil {
		return err
	}
	for _, f := range response.Features {
		if len(f.Geometry.Rings) > 0 {
			site.Footprints = append(site.Footprints, Feature{Source: "Seattle Building Outlines 2023", CRS: "EPSG:2926", Rings: rings(f.Geometry.Rings)})
		}
	}
	return nil
}
func (k KingCounty) elevation(ctx context.Context, site *Site) error {
	var response struct {
		Value json.Number `json:"value"`
	}
	params := url.Values{"geometry": {fmt.Sprintf("{\"x\":%f,\"y\":%f,\"spatialReference\":{\"wkid\":4326}}", site.Address.Location.X, site.Address.Location.Y)}, "geometryType": {"esriGeometryPoint"}, "sr": {"4326"}, "returnGeometry": {"false"}}
	if err := query(ctx, k.client(), elevationURL, params, &response); err != nil {
		return err
	}
	if v, err := response.Value.Float64(); err == nil {
		site.TerrainElevationM = &v
	}
	return nil
}
func rings(raw [][][]float64) [][]Point {
	out := make([][]Point, 0, len(raw))
	for _, ring := range raw {
		points := make([]Point, 0, len(ring))
		for _, p := range ring {
			if len(p) >= 2 {
				points = append(points, Point{p[0], p[1]})
			}
		}
		out = append(out, points)
	}
	return out
}
func aerialLink(x, y float64) string {
	q := url.Values{"bbox": {fmt.Sprintf("%f,%f,%f,%f", x-.001, y-.001, x+.001, y+.001)}, "bboxSR": {"4326"}, "imageSR": {"4326"}, "size": {"1024,1024"}, "format": {"jpg"}, "f": {"image"}}
	return aerialURL + "?" + q.Encode()
}
