package store

import (
	"github.com/feiyushaw/ztemate/internal/zte"
	"path/filepath"
	"testing"
)

func TestArchiveIdentityAndRouterIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	m := zte.Message{RemoteID: "1", Sender: "10010", Date: "26,11,09,12,00,00", Content: "first", Unread: true}
	if _, e = s.Save("http://router-a", []zte.Message{m}); e != nil {
		t.Fatal(e)
	}
	m.Unread = false
	s.Save("http://router-a", []zte.Message{m})
	rows, e := s.List("http://router-a")
	if e != nil || len(rows) != 1 || rows[0].Unread {
		t.Fatalf("upsert %#v %v", rows, e)
	}
	m.Content = "reused id"
	s.Save("http://router-a", []zte.Message{m})
	s.Save("http://router-b", []zte.Message{m})
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, _ := s.List("http://router-a")
	b, _ := s.List("http://router-b")
	if len(a) != 2 || len(b) != 1 {
		t.Fatalf("ID reuse/isolation: %d/%d", len(a), len(b))
	}
}
