package ident

import (
	"net/http"
	"testing"
)

func TestNormalizeStableAndUUIDPassthrough(t *testing.T) {
	t.Parallel()
	u := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	if Normalize(u).String() != u {
		t.Fatalf("uuid should pass through")
	}
	a := Normalize("alice")
	b := Normalize("alice")
	if a != b {
		t.Fatal("same key must hash to the same UUID")
	}
	if Normalize("bob") == a {
		t.Fatal("different keys should differ")
	}
	if Normalize("dev:"+u).String() != u {
		t.Fatal("dev: prefix should strip")
	}
}

func TestFromRequestHeaderAndBearer(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest(http.MethodGet, "http://x/rooms", nil)
	req.Header.Set(HeaderPlayerID, "alice")
	req.Header.Set(HeaderPlayerName, "Alice")
	p, err := FromRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != Normalize("alice").String() || p.DisplayName != "Alice" {
		t.Fatalf("%+v", p)
	}

	req2, _ := http.NewRequest(http.MethodGet, "http://x/ws?token=alice", nil)
	p2, err := FromRequest(req2)
	if err != nil {
		t.Fatal(err)
	}
	if p2.ID != p.ID {
		t.Fatal("query token should match header")
	}

	req3, _ := http.NewRequest(http.MethodGet, "http://x/rooms", nil)
	if _, err := FromRequest(req3); err != ErrNoPlayer {
		t.Fatalf("want ErrNoPlayer, got %v", err)
	}

	req4, _ := http.NewRequest(http.MethodGet, "http://x/rooms", nil)
	req4.Header.Set("Authorization", "Bearer header.payload.sig")
	if _, err := FromRequest(req4); err != ErrNoPlayer {
		t.Fatalf("JWT bearer must not become a hashed guest, got %v", err)
	}
}
