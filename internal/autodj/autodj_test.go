package autodj

import (
	"testing"
)

// --- MoodGraph integrity ---

func TestAllGenresHaveAdjacent(t *testing.T) {
	for name, g := range MoodGraph {
		if len(g.Adjacent) == 0 {
			t.Errorf("Genre %q has no adjacent genres", name)
		}
	}
}

func TestAdjacencyIsSymmetric(t *testing.T) {
	for name, g := range MoodGraph {
		for _, adj := range g.Adjacent {
			neighbor, ok := MoodGraph[adj]
			if !ok {
				t.Errorf("Genre %q lists non-existent adjacent genre %q", name, adj)
				continue
			}
			found := false
			for _, back := range neighbor.Adjacent {
				if back == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Asymmetric edge: %q -> %q exists, but %q -> %q does not", name, adj, adj, name)
			}
		}
	}
}

func TestGraphIsFullyConnected(t *testing.T) {
	if len(MoodGraph) == 0 {
		t.Fatal("MoodGraph is empty")
	}

	// BFS from an arbitrary start node
	var start string
	for name := range MoodGraph {
		start = name
		break
	}

	visited := map[string]bool{start: true}
	queue := []string{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, adj := range MoodGraph[current].Adjacent {
			if !visited[adj] {
				visited[adj] = true
				queue = append(queue, adj)
			}
		}
	}

	if len(visited) != len(MoodGraph) {
		unreachable := []string{}
		for name := range MoodGraph {
			if !visited[name] {
				unreachable = append(unreachable, name)
			}
		}
		t.Errorf("Graph not fully connected from %q. Unreachable: %v", start, unreachable)
	}
}

func TestGenreCount(t *testing.T) {
	if got := len(MoodGraph); got != 14 {
		t.Errorf("Expected 14 genres, got %d", got)
	}
}

func TestGenreNameConsistency(t *testing.T) {
	for name, g := range MoodGraph {
		if g.Name != name {
			t.Errorf("Genre map key %q != Genre.Name %q", name, g.Name)
		}
	}
}

// --- GenreNames ---

func TestGenreNames(t *testing.T) {
	names := GenreNames()
	if len(names) != len(MoodGraph) {
		t.Errorf("GenreNames() returned %d names, want %d", len(names), len(MoodGraph))
	}

	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			t.Errorf("Duplicate genre name: %q", name)
		}
		seen[name] = true
		if !IsValidGenre(name) {
			t.Errorf("GenreNames() returned %q but IsValidGenre says false", name)
		}
	}
}

// --- IsValidGenre ---

func TestIsValidGenre(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"ambient", true},
		{"lofi hip hop", true},
		{"drum and bass", true},
		{"metal", false},
		{"", false},
		{"Ambient", false}, // case sensitive
	}
	for _, tt := range tests {
		if got := IsValidGenre(tt.name); got != tt.want {
			t.Errorf("IsValidGenre(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// --- Captions ---

func TestAllGenresHaveCaptions(t *testing.T) {
	for name := range MoodGraph {
		caption := GetCaption(name)
		if caption == "" {
			t.Errorf("Genre %q has empty caption", name)
		}
		// Captions should be meaningful (at least 20 chars)
		if len(caption) < 20 {
			t.Errorf("Genre %q caption too short (%d chars): %q", name, len(caption), caption)
		}
	}
}

func TestGetCaptionKnownGenre(t *testing.T) {
	caption := GetCaption("jazz")
	if caption == "" {
		t.Fatal("GetCaption(jazz) returned empty")
	}
	// Jazz caption should reference jazz-related terms
	if !containsAny(caption, "jazz", "piano", "bass", "swing") {
		t.Errorf("Jazz caption seems wrong: %q", caption)
	}
}

func TestGetCaptionUnknownGenre(t *testing.T) {
	caption := GetCaption("polka")
	if caption == "" {
		t.Fatal("GetCaption(polka) returned empty for unknown genre")
	}
	// Should use the fallback template
	if !containsAny(caption, "polka") {
		t.Errorf("Unknown genre fallback should include genre name: %q", caption)
	}
}

func TestCaptionsAreInstrumental(t *testing.T) {
	// Phase 1: no lyrics. Captions shouldn't contain vocal keywords.
	vocalsKeywords := []string{"sing", "vocal", "lyrics", "voice", "rapper", "verse", "chorus"}
	for name := range MoodGraph {
		caption := GetCaption(name)
		for _, kw := range vocalsKeywords {
			if containsWord(caption, kw) {
				t.Errorf("Genre %q caption contains vocal keyword %q (Phase 1 is instrumental only): %q", name, kw, caption)
			}
		}
	}
}

// --- TrackName ---

func TestTrackNameKnownGenre(t *testing.T) {
	name := TrackName("jazz", "abc12345-def6-7890")
	if name == "" {
		t.Fatal("TrackName returned empty for known genre")
	}
	if !contains(name, "jazz") {
		t.Errorf("TrackName should contain genre: got %q", name)
	}
}

func TestTrackNameDeterministic(t *testing.T) {
	a := TrackName("ambient", "test-id-001")
	b := TrackName("ambient", "test-id-001")
	if a != b {
		t.Errorf("TrackName not deterministic: %q != %q", a, b)
	}
}

func TestTrackNameEmpty(t *testing.T) {
	if TrackName("", "some-id") != "" {
		t.Error("TrackName should return empty for empty genre")
	}
	if TrackName("jazz", "") != "" {
		t.Error("TrackName should return empty for empty trackID")
	}
}

func TestTrackNameUnknownGenre(t *testing.T) {
	name := TrackName("polka", "some-id")
	if name != "polka session" {
		t.Errorf("TrackName for unknown genre should be 'polka session', got %q", name)
	}
}

func TestAllGenresHaveAdjectives(t *testing.T) {
	for name := range MoodGraph {
		adjs := genreAdjectives[name]
		if len(adjs) == 0 {
			t.Errorf("Genre %q has no adjectives for track naming", name)
		}
	}
}

// --- SchedulerConfig defaults ---

func TestSchedulerConfigDefaults(t *testing.T) {
	cfg := SchedulerConfig{
		StartingGenre:  "lofi hip hop",
		TrackDuration:  180,
		BufferAhead:    3,
		DwellMin:       300,
		DwellMax:       900,
		InferenceSteps: 8,
	}

	if !IsValidGenre(cfg.StartingGenre) {
		t.Errorf("Default starting genre %q not in mood graph", cfg.StartingGenre)
	}
	if cfg.DwellMin >= cfg.DwellMax {
		t.Errorf("DwellMin (%d) >= DwellMax (%d)", cfg.DwellMin, cfg.DwellMax)
	}
	if cfg.BufferAhead < 1 {
		t.Errorf("BufferAhead should be >= 1, got %d", cfg.BufferAhead)
	}
}

// --- helpers ---

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func containsWord(s, word string) bool {
	// Simple word boundary check
	for i := 0; i <= len(s)-len(word); i++ {
		if s[i:i+len(word)] == word {
			// Check boundaries
			before := i == 0 || s[i-1] == ' ' || s[i-1] == ','
			after := i+len(word) == len(s) || s[i+len(word)] == ' ' || s[i+len(word)] == ','
			if before && after {
				return true
			}
		}
	}
	return false
}
