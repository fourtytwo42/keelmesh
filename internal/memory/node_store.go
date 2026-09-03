package memory

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"go.etcd.io/bbolt"
	_ "modernc.org/sqlite"
)

type localStore struct {
	db      *sql.DB
	journal *bbolt.DB
}

func openLocalStore(root string) (*localStore, error) {
	if root == "" {
		root = "/var/lib/keelmesh-node/memory"
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(root, "memory.sqlite"))
	if err != nil {
		return nil, err
	}
	statements := []string{"PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA foreign_keys=ON", "CREATE TABLE IF NOT EXISTS turns(id TEXT PRIMARY KEY,payload BLOB NOT NULL,created_at TEXT NOT NULL)", "CREATE TABLE IF NOT EXISTS scenes(id TEXT PRIMARY KEY,payload BLOB NOT NULL,pinned INTEGER NOT NULL,updated_at TEXT NOT NULL)", "CREATE TABLE IF NOT EXISTS items(id TEXT PRIMARY KEY,payload BLOB NOT NULL,tombstoned INTEGER NOT NULL,updated_at TEXT NOT NULL)", "CREATE TABLE IF NOT EXISTS candidates(id TEXT PRIMARY KEY,payload BLOB NOT NULL,state TEXT NOT NULL,created_at TEXT NOT NULL)", "CREATE TABLE IF NOT EXISTS retrieval_metadata(id TEXT PRIMARY KEY,payload BLOB NOT NULL,created_at TEXT NOT NULL)"}
	for _, statement := range statements {
		if _, err = db.Exec(statement); err != nil {
			db.Close()
			return nil, err
		}
	}
	journal, err := bbolt.Open(filepath.Join(root, "memory-journal.bbolt"), 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = journal.Update(func(tx *bbolt.Tx) error { _, e := tx.CreateBucketIfNotExists([]byte("events")); return e }); err != nil {
		journal.Close()
		db.Close()
		return nil, err
	}
	return &localStore{db: db, journal: journal}, nil
}
func (s *localStore) Close() error { _ = s.journal.Sync(); _ = s.journal.Close(); return s.db.Close() }
func (s *localStore) putTurn(v domain.ConversationTurnV1) error {
	raw, _ := json.Marshal(v)
	if _, err := s.db.Exec("INSERT OR IGNORE INTO turns(id,payload,created_at) VALUES(?,?,?)", v.ID, raw, v.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.append("turn", v.ID, raw)
}
func (s *localStore) putItem(v domain.MemoryItemV1) error {
	raw, _ := json.Marshal(v)
	if _, err := s.db.Exec("INSERT INTO items(id,payload,tombstoned,updated_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload,tombstoned=excluded.tombstoned,updated_at=excluded.updated_at", v.ID, raw, v.Tombstoned, v.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.append("item", v.ID, raw)
}
func (s *localStore) putCandidate(v domain.MemoryCandidateV1) error {
	raw, _ := json.Marshal(v)
	if _, err := s.db.Exec("INSERT INTO candidates(id,payload,state,created_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload,state=excluded.state", v.ID, raw, v.State, v.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.append("candidate", v.ID, raw)
}
func (s *localStore) putScene(v domain.CommandSceneV1) error {
	raw, _ := json.Marshal(v)
	if _, err := s.db.Exec("INSERT INTO scenes(id,payload,pinned,updated_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload,pinned=excluded.pinned,updated_at=excluded.updated_at", v.ID, raw, v.Pinned, v.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.append("scene", v.ID, raw)
}
func (s *localStore) tombstone(v domain.MemoryItemV1) error { return s.putItem(v) }
func (s *localStore) append(kind, id string, raw []byte) error {
	return s.journal.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("events"))
		sequence, _ := bucket.NextSequence()
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, sequence)
		value, _ := json.Marshal(map[string]any{"kind": kind, "id": id, "payload": json.RawMessage(raw), "created_at": time.Now().UTC()})
		return bucket.Put(key, value)
	})
}
func (s *localStore) load() ([]domain.MemoryItemV1, []domain.ConversationTurnV1, []domain.MemoryCandidateV1, []domain.CommandSceneV1, error) {
	items := []domain.MemoryItemV1{}
	turns := []domain.ConversationTurnV1{}
	candidates := []domain.MemoryCandidateV1{}
	scenes := []domain.CommandSceneV1{}
	rows, err := s.db.Query("SELECT payload FROM items ORDER BY updated_at")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var raw []byte
		var v domain.MemoryItemV1
		if rows.Scan(&raw) == nil && json.Unmarshal(raw, &v) == nil {
			items = append(items, v)
		}
	}
	rows.Close()
	rows, err = s.db.Query("SELECT payload FROM turns ORDER BY created_at DESC LIMIT 2000")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var raw []byte
		var v domain.ConversationTurnV1
		if rows.Scan(&raw) == nil && json.Unmarshal(raw, &v) == nil {
			turns = append(turns, v)
		}
	}
	rows.Close()
	reverseTurns(turns)
	rows, err = s.db.Query("SELECT payload FROM candidates ORDER BY created_at")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var raw []byte
		var v domain.MemoryCandidateV1
		if rows.Scan(&raw) == nil && json.Unmarshal(raw, &v) == nil {
			candidates = append(candidates, v)
		}
	}
	rows.Close()
	rows, err = s.db.Query("SELECT payload FROM scenes ORDER BY updated_at DESC LIMIT 200")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var raw []byte
		var v domain.CommandSceneV1
		if rows.Scan(&raw) == nil && json.Unmarshal(raw, &v) == nil {
			scenes = append(scenes, v)
		}
	}
	rows.Close()
	return items, turns, candidates, scenes, nil
}
