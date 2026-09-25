# Design Public

Design Public is a small, public-data site-study engine. Given a supported
county address it returns a parcel outline, public building footprints,
best-effort terrain elevation, and an aerial-image URL. It also includes a
YAML-driven residential design-stage checker and a stdio MCP server.

Supported counties: King County, WA.

Maintenance: slow until 2027-01.

## Run

```sh
docker compose up
```

Open `http://localhost:8080` and enter a King County address. The form starts
with **500 4th Ave, Seattle, WA 98104**, a public civic-address example; do not
enter private addresses into services you do not control.

Without Docker:

```sh
go run ./cmd/design-public
go run . --rules rules/seattle-residential.yml examples/synthetic-room.yml
go run ./cmd/mcp
```

Bundled: Seattle residential (11 rules, each tagged with a confidence tier and citation).
Verify against the adopted code before relying on a finding. The generic schema
template remains available at `rules/template.yml`. See
[`docs/COUNTY-ADAPTERS.md`](docs/COUNTY-ADAPTERS.md) to add a county.

## Privacy and scope

The repository intentionally contains no addresses belonging to the project
owner, photos, LiDAR tiles, renders, assessor exports, or financial records.
Lookups query public services at request time; no LiDAR processing or imagery
is stored by this project.
