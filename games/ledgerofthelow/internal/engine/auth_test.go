package engine

import (
	"bytes"
	"strings"
	"testing"

	"interdoor.net/interdoor/internal/engine/term"
)

func TestAuthRoundTrip(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateAccount("Maren", "hookline")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.HasPrefix(p.GlobalID, "node01:p_") {
		t.Fatalf("bad global id: %q", p.GlobalID)
	}
	if _, err := s.Authenticate("Maren", "hookline"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := s.Authenticate("maren", "hookline"); err != nil {
		t.Fatalf("case-insensitive login: %v", err)
	}
	if _, err := s.Authenticate("Maren", "wrong"); err == nil {
		t.Fatalf("wrong password should fail")
	}
	if _, err := s.CreateAccount("Maren", "other"); err == nil {
		t.Fatalf("duplicate name should fail")
	}
}

func TestPlayerCreatedEvent(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateAccount("Thursen", "stories"); err != nil {
		t.Fatalf("create: %v", err)
	}
	var n int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM events WHERE type='player.created'`).Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 player.created event, got %d", n)
	}
}

func TestNameValidation(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateAccount("ab", "secret"); err == nil {
		t.Fatalf("short name should fail")
	}
	if _, err := s.CreateAccount("Valid", "x"); err == nil {
		t.Fatalf("short password should fail")
	}
}

func TestLoginRejectsTravelingPlayer(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateAccount("Traveler", "roadpass")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.SetPlayerStatus(p.GlobalID, "traveling"); err != nil {
		t.Fatalf("set traveling: %v", err)
	}

	var out bytes.Buffer
	tty := term.New(strings.NewReader("Traveler\nroadpass\n"), &out)
	srv := &Server{store: s}
	if got := srv.doLogin(tty); got != nil {
		t.Fatalf("traveling player should not log in, got %+v", got)
	}
	if !strings.Contains(out.String(), "not here right now") {
		t.Fatalf("traveling login message missing from output: %q", out.String())
	}
}
