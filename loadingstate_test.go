package main

import (
	"testing"

	"github.com/carlmjohnson/be"
)

func TestNewLoadingState(t *testing.T) {
	tests := []struct {
		name string
		keys []string
	}{
		{
			name: "empty keys",
			keys: []string{},
		},
		{
			name: "single key",
			keys: []string{"test"},
		},
		{
			name: "multiple keys",
			keys: []string{"categories", "transactions", "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := newLoadingState(tt.keys...)

			// Test that all keys are initialized to false
			for _, key := range tt.keys {
				value, exists := ls[key]
				be.True(t, exists)
				be.False(t, value)
			}

			// Test length
			be.Equal(t, len(tt.keys), len(ls))
		})
	}
}

func TestLoadingStateSet(t *testing.T) {
	ls := newLoadingState("test1", "test2")

	// Initially should be false
	be.False(t, ls["test1"])
	be.False(t, ls["test2"])

	// Set one key
	ls.set("test1")
	be.True(t, ls["test1"])
	be.False(t, ls["test2"])

	// Set another key
	ls.set("test2")
	be.True(t, ls["test1"])
	be.True(t, ls["test2"])
}

func TestLoadingStateUnset(t *testing.T) {
	ls := newLoadingState("test1", "test2")

	// Set both keys
	ls.set("test1")
	ls.set("test2")
	be.True(t, ls["test1"])
	be.True(t, ls["test2"])

	// Unset one key
	ls.unset("test1")
	be.False(t, ls["test1"])
	be.True(t, ls["test2"])

	// Unset another key
	ls.unset("test2")
	be.False(t, ls["test1"])
	be.False(t, ls["test2"])
}

func TestLoadingStateAllLoaded(t *testing.T) {
	tests := []struct {
		name            string
		keys            []string
		setKeys         []string
		expectLoaded    bool
		expectNotLoaded string
	}{
		{
			name:            "empty state - all loaded",
			keys:            []string{},
			setKeys:         []string{},
			expectLoaded:    true,
			expectNotLoaded: "",
		},
		{
			name:            "none loaded",
			keys:            []string{"test1", "test2"},
			setKeys:         []string{},
			expectLoaded:    false,
			expectNotLoaded: "test1", // or "test2" - map iteration order
		},
		{
			name:            "partially loaded",
			keys:            []string{"test1", "test2", "test3"},
			setKeys:         []string{"test1", "test3"},
			expectLoaded:    false,
			expectNotLoaded: "test2",
		},
		{
			name:            "all loaded",
			keys:            []string{"test1", "test2", "test3"},
			setKeys:         []string{"test1", "test2", "test3"},
			expectLoaded:    true,
			expectNotLoaded: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := newLoadingState(tt.keys...)

			// Set specified keys
			for _, key := range tt.setKeys {
				ls.set(key)
			}

			loaded, notLoaded := ls.allLoaded()
			be.Equal(t, tt.expectLoaded, loaded)

			if tt.expectLoaded {
				be.Equal(t, "", notLoaded)
			} else {
				// Should return one of the not-loaded keys
				be.Nonzero(t, notLoaded)
				// Verify it's actually not loaded
				be.False(t, ls[notLoaded])
			}
		})
	}
}
