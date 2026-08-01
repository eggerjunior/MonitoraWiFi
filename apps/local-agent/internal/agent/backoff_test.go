package agent

import (
	"testing"
	"time"
)

func TestBackoff_DoublesUntilMax(t *testing.T) {
	b := NewBackoff(time.Second, 8*time.Second)

	got := []time.Duration{b.Next(), b.Next(), b.Next(), b.Next(), b.Next()}
	want := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		8 * time.Second, // teto atingido, não passa disso
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tentativa %d: esperava %v, recebeu %v", i, want[i], got[i])
		}
	}
}

func TestBackoff_ResetReturnsToInitial(t *testing.T) {
	b := NewBackoff(time.Second, time.Minute)
	b.Next()
	b.Next()
	b.Reset()

	if got := b.Next(); got != time.Second {
		t.Fatalf("esperava voltar ao valor inicial após Reset, recebeu %v", got)
	}
}
