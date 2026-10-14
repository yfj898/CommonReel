package workflow

import (
	"context"
	"encoding/json"
	"time"
)

// Supplier metadata stays in the local, ignored database; never package it.
func (a *App) loadSupply(ctx context.Context) error {
	if _, e := a.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS v2_supply(film_id TEXT PRIMARY KEY,payload TEXT NOT NULL)`); e != nil {
		return e
	}
	rows, e := a.db.QueryContext(ctx, "SELECT payload FROM v2_supply")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var b string
		if e = rows.Scan(&b); e != nil {
			return e
		}
		var av Availability
		if json.Unmarshal([]byte(b), &av) == nil && a.cfg.Now().Sub(av.FetchedAt) < 30*24*time.Hour {
			av.Version = availabilityVersion(av)
			a.cache[av.FilmID] = av
		}
	}
	return rows.Err()
}
func (a *App) persistSupply(ctx context.Context, av Availability) {
	b, e := json.Marshal(av)
	if e == nil {
		a.db.ExecContext(ctx, "INSERT INTO v2_supply VALUES(?,?) ON CONFLICT(film_id) DO UPDATE SET payload=excluded.payload", av.FilmID, string(b))
	}
}
