package passcode

import (
	"regexp"
	"testing"
)

func TestNewFormat(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile(`^[0-9]{6}$`)
	seen := map[string]int{}
	for i := 0; i < 500; i++ {
		c := New()
		if !re.MatchString(c) {
			t.Fatalf("New() = %q, want 6 digits", c)
		}
		if !Valid(c) {
			t.Fatalf("Valid(%q) = false", c)
		}
		seen[c]++
	}
	if len(seen) < 100 {
		t.Fatalf("only %d unique codes in 500 draws; rng looks stuck", len(seen))
	}
}

func TestValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want bool
	}{
		{"000000", true},
		{"999999", true},
		{"482910", true},
		{"12345", false},
		{"1234567", false},
		{"12a456", false},
		{"", false},
		{" 123456", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.in); got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestAllocateRetriesOnCollision(t *testing.T) {
	t.Parallel()
	n := 0
	gen := func() string {
		n++
		if n < 3 {
			return "000000"
		}
		return "000001"
	}
	taken := map[string]bool{"000000": true}
	code, err := Allocate(func(c string) bool { return taken[c] }, gen, 10)
	if err != nil {
		t.Fatal(err)
	}
	if code != "000001" {
		t.Fatalf("got %q, want 000001 after colliding twice", code)
	}
	if n != 3 {
		t.Fatalf("gen called %d times, want 3", n)
	}
}

func TestAllocateExhausted(t *testing.T) {
	t.Parallel()
	_, err := Allocate(func(string) bool { return true }, func() string { return "123456" }, 3)
	if err != ErrExhausted {
		t.Fatalf("err = %v, want ErrExhausted", err)
	}
}

func TestAllocateSkipsInvalid(t *testing.T) {
	t.Parallel()
	n := 0
	gen := func() string {
		n++
		if n == 1 {
			return "nope"
		}
		return "424242"
	}
	code, err := Allocate(func(string) bool { return false }, gen, 5)
	if err != nil {
		t.Fatal(err)
	}
	if code != "424242" {
		t.Fatalf("got %q", code)
	}
}
