package audit

import (
	"bytes"
	"testing"
	"time"
)

func TestCanonicalHash(t *testing.T) {
	writer, _ := NewWriter(bytes.Repeat([]byte{7}, 32))
	stamp := time.Date(2026, 10, 11, 1, 2, 3, 456789000, time.UTC)
	event := Event{ChainID: "patient", Seq: 1, OccurredAt: stamp, ActorID: "doctor", Action: "episode.view", ResourceType: "care_episode", ResourceID: "episode", Outcome: "allow", Basis: "episode_doctor", Context: map[string]any{"b": "two", "a": "one"}, PrevHash: make([]byte, 32)}
	first, err := writer.digest(event)
	if err != nil {
		t.Fatal(err)
	}
	event.Context = map[string]any{"a": "one", "b": "two"}
	event.OccurredAt = stamp.In(time.FixedZone("UTC+7", 7*3600))
	second, _ := writer.digest(event)
	if !bytes.Equal(first, second) {
		t.Fatal("JSON key order/timezone changed canonical hash")
	}
	event.Outcome = "deny"
	changed, _ := writer.digest(event)
	if bytes.Equal(first, changed) {
		t.Fatal("payload manipulation undetected")
	}
	event.Outcome = "allow"
	event.PrevHash[0] = 1
	changed, _ = writer.digest(event)
	if bytes.Equal(first, changed) {
		t.Fatal("previous hash not included")
	}
	if _, err := NewWriter([]byte("short")); err == nil {
		t.Fatal("short key accepted")
	}
}

func TestKeyRotationVerification(t *testing.T) {
	oldKey := bytes.Repeat([]byte{1}, 32)
	newKey := bytes.Repeat([]byte{2}, 32)

	oldWriter, err := NewWriter(oldKey)
	if err != nil {
		t.Fatal(err)
	}

	event := Event{
		ChainID:      "patient-1",
		Seq:          1,
		OccurredAt:   time.Now().UTC(),
		ActorID:      "actor-1",
		Action:       "view",
		ResourceType: "record",
		ResourceID:   "rec-1",
		Outcome:      "allow",
		Basis:        "direct",
		Context:      map[string]any{},
		PrevHash:     make([]byte, 32),
	}
	hash, err := oldWriter.digest(event)
	if err != nil {
		t.Fatal(err)
	}
	event.Hash = hash

	// New writer without oldKey must fail verification.
	newWriterWithoutFallback, _ := NewWriter(newKey)
	if newWriterWithoutFallback.verifyDigest(event) {
		t.Fatal("expected digest verification to fail without historical key")
	}

	// New writer WITH oldKey as fallback must succeed.
	newWriterWithFallback, _ := NewWriter(newKey, oldKey)
	if !newWriterWithFallback.verifyDigest(event) {
		t.Fatal("expected digest verification to succeed with fallback key")
	}
	event.Action = "tampered"
	if newWriterWithFallback.verifyDigest(event) {
		t.Fatal("fallback accepted a modified event")
	}
	if _, err := NewWriter(newKey, []byte("short")); err == nil {
		t.Fatal("invalid historical key accepted")
	}
}
