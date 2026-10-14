package room

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var ErrConflict = errors.New("room changed; refresh and retry with its current revision")
var ErrNotFound = errors.New("room not found")

type Member struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Profile int      `json:"profile"`
	Likes   []string `json:"likes"`
	Vetoes  []string `json:"vetoes"`
}
type Choice struct {
	MovieID      int    `json:"movie_id"`
	Title        string `json:"title"`
	FromRevision int    `json:"from_revision"`
	SavedAt      string `json:"saved_at"`
	Explanation  string `json:"explanation"`
}
type Room struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Revision    int      `json:"revision"`
	Members     []Member `json:"members"`
	Choices     []Choice `json:"choices"`
	ExcludeSeen bool     `json:"exclude_seen"`
	UpdatedAt   string   `json:"updated_at"`
}
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func ID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"PRAGMA busy_timeout=5000", "PRAGMA journal_mode=WAL", `CREATE TABLE IF NOT EXISTS rooms(id TEXT PRIMARY KEY,revision INTEGER NOT NULL,payload TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS operations(room_id TEXT NOT NULL,op_key TEXT NOT NULL,request_hash TEXT NOT NULL,response TEXT NOT NULL,PRIMARY KEY(room_id,op_key))`} {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			return nil, err
		}
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Create(ctx context.Context, name string) (Room, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return Room{}, fmt.Errorf("room name must contain 1–80 bytes")
	}
	r := Room{ID: ID(), Name: name, Revision: 1, Members: []Member{}, Choices: []Choice{}, ExcludeSeen: true, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	b, _ := json.Marshal(r)
	_, err := s.db.ExecContext(ctx, "INSERT INTO rooms VALUES(?,?,?)", r.ID, r.Revision, string(b))
	return r, err
}
func (s *Store) Get(ctx context.Context, id string) (Room, error) {
	var b string
	err := s.db.QueryRowContext(ctx, "SELECT payload FROM rooms WHERE id=?", id).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return Room{}, ErrNotFound
	}
	if err != nil {
		return Room{}, err
	}
	var r Room
	err = json.Unmarshal([]byte(b), &r)
	return r, err
}

// Mutate atomically binds an operation key to its exact request and result.
// Replaying the same request returns the original result, before checking CAS.
func (s *Store) Mutate(ctx context.Context, id string, revision int, key string, request any, change func(*Room) error) (Room, error) {
	if len(key) < 8 || len(key) > 128 {
		return Room{}, fmt.Errorf("operation key must contain 8–128 bytes")
	}
	b, err := json.Marshal(request)
	if err != nil {
		return Room{}, err
	}
	h := sha256.Sum256(b)
	hash := hex.EncodeToString(h[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Room{}, err
	}
	defer tx.Rollback()
	var oldHash, response string
	err = tx.QueryRowContext(ctx, "SELECT request_hash,response FROM operations WHERE room_id=? AND op_key=?", id, key).Scan(&oldHash, &response)
	if err == nil {
		if oldHash != hash {
			return Room{}, fmt.Errorf("operation key reused for a different request")
		}
		var r Room
		err = json.Unmarshal([]byte(response), &r)
		return r, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Room{}, err
	}
	var payload string
	err = tx.QueryRowContext(ctx, "SELECT payload FROM rooms WHERE id=?", id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return Room{}, ErrNotFound
	}
	if err != nil {
		return Room{}, err
	}
	var r Room
	if err = json.Unmarshal([]byte(payload), &r); err != nil {
		return Room{}, err
	}
	if r.Revision != revision {
		return Room{}, ErrConflict
	}
	if err = change(&r); err != nil {
		return Room{}, err
	}
	r.Revision++
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	b, err = json.Marshal(r)
	if err != nil {
		return Room{}, err
	}
	res, err := tx.ExecContext(ctx, "UPDATE rooms SET payload=?,revision=? WHERE id=? AND revision=?", string(b), r.Revision, id, revision)
	if err != nil {
		return Room{}, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return Room{}, ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operations VALUES(?,?,?,?)", id, key, hash, string(b)); err != nil {
		return Room{}, err
	}
	if err = tx.Commit(); err != nil {
		return Room{}, err
	}
	return r, nil
}
