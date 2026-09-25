// design-public serves a browser-ready, public-data site-study API.
package main

import (
	"design-public/internal/county"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	adapter := county.KingCounty{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("GET /api/site", func(w http.ResponseWriter, r *http.Request) {
		site, err := adapter.Lookup(r.Context(), r.URL.Query().Get("address"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, site)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, page)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("design-public listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

const page = `<!doctype html><title>Design Public</title><meta name="viewport" content="width=device-width,initial-scale=1"><main><h1>Public site study</h1><p>King County, WA public-data preview.</p><form><label>Address <input name="address" value="500 4th Ave, Seattle, WA 98104" required></label><button>Look up</button></form><pre id="result"></pre></main><script>const f=document.querySelector('form'),o=document.querySelector('#result');f.onsubmit=async e=>{e.preventDefault();o.textContent='Loading…';const r=await fetch('/api/site?'+new URLSearchParams(new FormData(f)));o.textContent=JSON.stringify(await r.json(),null,2)}</script>`
