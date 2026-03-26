package handler

import (
	"testing"
	"time"
)

func TestSessionCreate(t *testing.T) {
	store := NewSessionStore(15 * time.Minute)

	token := store.Create("0xAlice")

	if len(token) != 64 {
		t.Fatalf("expected 64-char hex token, got %d chars: %s", len(token), token)
	}

	addr, ok := store.Validate(token)
	if !ok {
		t.Fatal("expected token to be valid")
	}
	if addr != "0xAlice" {
		t.Fatalf("expected address 0xAlice, got %s", addr)
	}
}

func TestSessionValidateUnknownToken(t *testing.T) {
	store := NewSessionStore(15 * time.Minute)

	_, ok := store.Validate("nonexistent")
	if ok {
		t.Fatal("expected unknown token to be invalid")
	}
}

func TestSessionExpiry(t *testing.T) {
	store := NewSessionStore(1 * time.Millisecond)

	token := store.Create("0xBob")
	time.Sleep(5 * time.Millisecond)

	_, ok := store.Validate(token)
	if ok {
		t.Fatal("expected expired token to be invalid")
	}
}

func TestSessionUniqueness(t *testing.T) {
	store := NewSessionStore(15 * time.Minute)

	token1 := store.Create("0xAlice")
	token2 := store.Create("0xAlice")

	if token1 == token2 {
		t.Fatal("expected unique tokens for separate calls")
	}
}
