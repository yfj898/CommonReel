package workflow

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed web/*
var webAssets embed.FS

func (a *App) webRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, map[string]any{"invite_origin": a.cfg.InviteOrigin, "region": "US", "max_members": 4, "protocol": "2025-11-25"})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
		defer cancel()
		db := "ready"
		if a.db.PingContext(ctx) != nil {
			db = "unavailable"
		}
		cat := "ready"
		if len(a.cfg.Films) < 200 || len(a.cfg.Films) > 300 {
			cat = "incomplete"
		}
		availability := map[string]any{"status": "unconfigured"}
		if v, ok := a.cfg.Supplier.(*Watchmode); ok {
			availability = v.Status()
			availability["status"] = "configured; live facts checked per movie"
			if v.Key == "" {
				availability["status"] = "unconfigured"
			}
		} else if a.cfg.Supplier != nil {
			availability["status"] = "external adapter configured"
		}
		endpoint := a.cfg.OllamaEndpoint
		if endpoint == "" {
			endpoint = "http://127.0.0.1:11434"
		}
		model := "unavailable"
		req, _ := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(endpoint, "/")+"/api/tags", nil)
		if req != nil {
			if res, e := http.DefaultClient.Do(req); e == nil {
				var body struct {
					Models []struct {
						Name string `json:"name"`
					} `json:"models"`
				}
				json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body)
				res.Body.Close()
				for _, m := range body.Models {
					if m.Name == a.cfg.OllamaModel {
						model = "installed; tool performance requires separate evidence"
					}
				}
			}
		}
		write(w, 200, map[string]any{"app": "ready", "db": db, "catalog": map[string]any{"status": cat, "count": len(a.cfg.Films)}, "availability": availability, "model": map[string]any{"status": model, "name": a.cfg.OllamaModel}, "evidence_notice": "Readiness does not prove a completed workflow, real phone connectivity or user outcomes."})
	})
	root, _ := fs.Sub(webAssets, "web")
	mux.Handle("/", http.FileServer(http.FS(root)))
}
