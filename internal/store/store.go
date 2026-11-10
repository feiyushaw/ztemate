package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/feiyushaw/ztemate/internal/zte"
	_ "modernc.org/sqlite"
)

type Message struct {
	ID string `json:"id"`
	zte.Message
	ArchivedAt string `json:"archivedAt"`
}
type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	f.Close()
	if e = os.Chmod(path, 0600); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS messages (id TEXT PRIMARY KEY, router TEXT NOT NULL, payload TEXT NOT NULL, archived_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS documents (key TEXT PRIMARY KEY,payload TEXT NOT NULL); CREATE TABLE IF NOT EXISTS settings (id INTEGER PRIMARY KEY CHECK(id=1), payload TEXT NOT NULL);`)
	if e != nil {
		db.Close()
		return nil, e
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func identity(router string, m zte.Message) string {
	b, _ := json.Marshal([]string{router, m.RemoteID, m.Sender, m.Date, m.Content})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s *Store) Save(router string, ms []zte.Message) (int, error) {
	tx, e := s.db.Begin()
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	added := 0
	// Router pages are newest first; insert older records first for stable archive ordering.
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		id := identity(router, m)
		payload, _ := json.Marshal(m)
		r, err := tx.Exec(`INSERT INTO messages(id,router,payload,archived_at) VALUES(?,?,?,?) ON CONFLICT(id) DO NOTHING`, id, router, string(payload), time.Now().Format(time.RFC3339Nano))
		if err != nil {
			return 0, err
		}
		n, err := r.RowsAffected()
		if err != nil {
			return 0, err
		}
		added += int(n)
		if _, err = tx.Exec(`UPDATE messages SET payload=? WHERE id=?`, string(payload), id); err != nil {
			return 0, err
		}
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	return added, nil
}
func (s *Store) List(router string) ([]Message, error) {
	rows, e := s.db.Query(`SELECT id,payload,archived_at FROM messages WHERE router=? ORDER BY archived_at DESC,rowid DESC`, router)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		var p string
		if e = rows.Scan(&m.ID, &p, &m.ArchivedAt); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(p), &m.Message); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) LoadSettings(v any) error {
	var p string
	e := s.db.QueryRow(`SELECT payload FROM settings WHERE id=1`).Scan(&p)
	if e == sql.ErrNoRows {
		return nil
	}
	if e != nil {
		return e
	}
	return json.Unmarshal([]byte(p), v)
}
func (s *Store) SaveSettings(v any) error {
	p, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = s.db.Exec(`INSERT INTO settings VALUES(1,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload`, string(p))
	return e
}

func (s *Store) LoadDocument(key string, v any) error {
	var p string
	e := s.db.QueryRow(`SELECT payload FROM documents WHERE key=?`, key).Scan(&p)
	if e == sql.ErrNoRows {
		return nil
	}
	if e != nil {
		return e
	}
	return json.Unmarshal([]byte(p), v)
}
func (s *Store) SaveDocument(key string, v any) error {
	p, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = s.db.Exec(`INSERT INTO documents VALUES(?,?) ON CONFLICT(key) DO UPDATE SET payload=excluded.payload`, key, string(p))
	return e
}
