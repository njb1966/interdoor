package engine

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrBadCredentials is returned when a login name/password pair does not match.
var ErrBadCredentials = fmt.Errorf("bad credentials")

// CreateAccount registers a new player. The display name must be 3-16 printable
// characters. The password is bcrypt-hashed. Emits a player.created event.
func (s *Store) CreateAccount(name, password string) (*Player, error) {
	name = strings.TrimSpace(name)
	if err := validName(name); err != nil {
		return nil, err
	}
	if len(password) < 4 {
		return nil, fmt.Errorf("password must be at least 4 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	p := &Player{
		GlobalID:  fmt.Sprintf("%s:p_%s", s.nodeID, newID()),
		Name:      name,
		HomeNode:  s.nodeID,
		Level:     1,
		Standing:  0,
		Status:    "active",
		CreatedAt: now,
		LastSeen:  now,
	}
	if err := s.InsertPlayer(p, string(hash)); err != nil {
		return nil, err
	}
	_ = s.Emit("player.created", map[string]any{
		"global_id":  p.GlobalID,
		"name":       p.Name,
		"home_node":  p.HomeNode,
		"created_at": p.CreatedAt.Unix(),
	})
	return p, nil
}

// Authenticate verifies a name/password and returns the player.
func (s *Store) Authenticate(name, password string) (*Player, error) {
	p, hash, err := s.PlayerByName(strings.TrimSpace(name))
	if err != nil {
		return nil, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, ErrBadCredentials
	}
	return p, nil
}

func validName(name string) error {
	if len(name) < 3 || len(name) > 16 {
		return fmt.Errorf("name must be 3-16 characters")
	}
	for _, r := range name {
		if r < 0x20 || r >= 0x7f {
			return fmt.Errorf("name must be printable ASCII")
		}
	}
	return nil
}

// newID returns a short lowercase base32 token for the local part of a player ID.
func newID() string {
	var b [5]byte
	_, _ = rand.Read(b[:])
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]))
}
