package hlc_test

import (
	"testing"
	"time"

	"github.com/laplasd/inforo/hlc"
)

func TestHLC_Now(t *testing.T) {
	t.Run("monotonic increment", func(t *testing.T) {
		clock := hlc.NewHLC()

		prev := clock.Now()
		for i := 0; i < 100; i++ {
			curr := clock.Now()
			if curr.Before(prev) {
				t.Fatalf("Clock moved backwards: %v < %v", curr, prev)
			}
			prev = curr
		}
	})

	t.Run("logical counter increments", func(t *testing.T) {
		clock := hlc.NewHLC()

		// Force same physical time
		fakeTime := uint64(time.Now().UnixNano())
		clock.Physical = fakeTime

		t1 := clock.Now()
		t2 := clock.Now()

		if t1.UnixNano() != t2.UnixNano() {
			t.Error("Expected same physical time with different logical counters")
		}
	})
}

func TestHLC_Update(t *testing.T) {
	tests := []struct {
		name     string
		initial  uint64
		external time.Time
		expected uint64
	}{
		{
			name:     "external newer",
			initial:  100,
			external: time.Unix(0, 150),
			expected: 150,
		},
		{
			name:     "external equal",
			initial:  100,
			external: time.Unix(0, 100),
			expected: 100,
		},
		{
			name:     "external older",
			initial:  100,
			external: time.Unix(0, 50),
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := hlc.NewHLC()
			clock.Physical = tt.initial

			clock.Update(tt.external)

			if clock.Physical != tt.expected {
				t.Errorf("Expected physical %d, got %d", tt.expected, clock.Physical)
			}
		})
	}
}

func TestGenerateNodeID(t *testing.T) {
	t.Run("uniqueness", func(t *testing.T) {
		const iterations = 1000
		ids := make(map[string]struct{}, iterations)

		for i := 0; i < iterations; i++ {
			id := hlc.GenerateNodeID()
			if _, exists := ids[id]; exists {
				t.Fatalf("Duplicate ID generated: %s", id)
			}
			ids[id] = struct{}{}
		}
	})

	t.Run("format", func(t *testing.T) {
		id := hlc.GenerateNodeID()
		if len(id) < 10 { // Минимальная ожидаемая длина
			t.Errorf("Unexpected ID format: %s", id)
		}
	})
}

func TestHLC_EdgeCases(t *testing.T) {
	t.Run("zero time update", func(t *testing.T) {
		clock := hlc.NewHLC()
		clock.Update(time.Time{}) // Zero time

		if clock.Physical == 0 {
			t.Error("Clock should not accept zero time")
		}
	})

	t.Run("max uint64", func(t *testing.T) {
		clock := hlc.NewHLC()
		clock.Physical = ^uint64(0) - 1

		// Не должно быть паники при переполнении
		t1 := clock.Now()
		t2 := clock.Now()

		if t1.After(t2) {
			t.Error("Time should handle overflow correctly")
		}
	})
}

func TestHLC_StringRepresentation(t *testing.T) {
	clock := hlc.NewHLC()
	clock.Physical = 123456789
	clock.Logical = 42

	// Проверяем, что String() или форматирование работает
	str := clock.Now().String()
	if str == "" {
		t.Error("Unexpected string representation")
	}
}
