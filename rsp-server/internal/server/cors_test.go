package server

import "testing"

func TestOriginAllowed(t *testing.T) {
	allow := []string{"http://localhost:5173", "https://play.example.com"}
	if !OriginAllowed("", allow) {
		t.Fatal("empty origin (curl) must pass")
	}
	if !OriginAllowed("http://evil.example", nil) {
		t.Fatal("empty allowlist is dev: any origin")
	}
	if !OriginAllowed("https://play.example.com", allow) {
		t.Fatal("listed origin")
	}
	if OriginAllowed("http://evil.example", allow) {
		t.Fatal("unlisted origin")
	}
}

func TestParseOrigins(t *testing.T) {
	got := ParseOrigins(" https://a.example ,http://localhost:5173, ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "http://localhost:5173" {
		t.Fatalf("%v", got)
	}
	if ParseOrigins("") != nil {
		t.Fatalf("%v", ParseOrigins(""))
	}
}
