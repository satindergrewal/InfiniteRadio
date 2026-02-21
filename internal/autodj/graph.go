package autodj

// Genre represents a node in the mood graph.
type Genre struct {
	Name     string
	Adjacent []string
}

// MoodGraph maps genre names to their graph nodes with adjacency edges.
// Transitions only follow edges -- no jumping across the graph.
var MoodGraph = map[string]*Genre{
	"ambient": {
		Name:     "ambient",
		Adjacent: []string{"chillwave", "classical"},
	},
	"chillwave": {
		Name:     "chillwave",
		Adjacent: []string{"ambient", "lofi hip hop", "classical", "synthwave"},
	},
	"lofi hip hop": {
		Name:     "lofi hip hop",
		Adjacent: []string{"chillwave", "jazz"},
	},
	"jazz": {
		Name:     "jazz",
		Adjacent: []string{"lofi hip hop", "bossa nova", "acoustic folk"},
	},
	"bossa nova": {
		Name:     "bossa nova",
		Adjacent: []string{"jazz"},
	},
	"acoustic folk": {
		Name:     "acoustic folk",
		Adjacent: []string{"jazz"},
	},
	"classical": {
		Name:     "classical",
		Adjacent: []string{"ambient", "chillwave", "cinematic"},
	},
	"cinematic": {
		Name:     "cinematic",
		Adjacent: []string{"classical", "indie rock"},
	},
	"synthwave": {
		Name:     "synthwave",
		Adjacent: []string{"chillwave", "electronic", "indie rock"},
	},
	"electronic": {
		Name:     "electronic",
		Adjacent: []string{"synthwave", "drum and bass", "disco funk"},
	},
	"drum and bass": {
		Name:     "drum and bass",
		Adjacent: []string{"electronic"},
	},
	"disco funk": {
		Name:     "disco funk",
		Adjacent: []string{"electronic", "rock"},
	},
	"indie rock": {
		Name:     "indie rock",
		Adjacent: []string{"cinematic", "synthwave", "rock"},
	},
	"rock": {
		Name:     "rock",
		Adjacent: []string{"indie rock", "disco funk"},
	},
}

// GenreNames returns all genre names in the mood graph.
func GenreNames() []string {
	names := make([]string, 0, len(MoodGraph))
	for name := range MoodGraph {
		names = append(names, name)
	}
	return names
}

// IsValidGenre checks if a genre exists in the mood graph.
func IsValidGenre(name string) bool {
	_, ok := MoodGraph[name]
	return ok
}
