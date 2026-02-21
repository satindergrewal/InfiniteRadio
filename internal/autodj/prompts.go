package autodj

// captions maps each genre to an ACE-Step generation caption.
// Each caption is 15-25 words describing instruments, mood, tempo, and production style.
// All instrumental in Phase 1 -- no lyrics.
var captions = map[string]string{
	"ambient": "Ethereal ambient soundscape with soft synthesizer pads, gentle reverb, slow evolving textures, peaceful and meditative atmosphere, minimal percussion",

	"chillwave": "Warm chillwave with hazy synthesizers, soft drum machine beats, nostalgic lo-fi tape warble, dreamy summer vibes, relaxed tempo",

	"lofi hip hop": "Lofi hip hop beat with vinyl crackle, mellow jazz piano chords, soft boom bap drums, warm bass, rainy day study vibes",

	"jazz": "Smooth jazz trio with upright bass walking lines, brushed drum kit, warm piano improvisations, late night club atmosphere, medium swing tempo",

	"bossa nova": "Gentle bossa nova with nylon string guitar, soft brushed percussion, warm upright bass, tropical breeze mood, relaxed Brazilian rhythm",

	"acoustic folk": "Intimate acoustic folk with fingerpicked steel string guitar, soft harmonica accents, warm double bass, campfire storytelling mood, gentle tempo",

	"classical": "Elegant classical chamber music with string quartet, flowing melodies, delicate piano accompaniment, refined and contemplative mood, moderate adagio tempo",

	"cinematic": "Epic cinematic orchestral score with sweeping strings, powerful brass, timpani, building emotional intensity, dramatic and inspiring, wide stereo soundstage",

	"synthwave": "Retro synthwave with pulsing analog synthesizers, driving arpeggios, electronic drums, neon-lit 1980s nostalgia, energetic mid-tempo groove",

	"electronic": "Modern electronic music with crisp synthesizers, four on the floor kick, layered pads, atmospheric breakdowns, uplifting progressive energy, festival vibes",

	"drum and bass": "High energy drum and bass with fast breakbeat patterns, deep rolling sub bass, atmospheric synth pads, dark warehouse rave atmosphere, 174 BPM",

	"disco funk": "Groovy disco funk with rhythmic guitar scratching, punchy bass slaps, tight horn stabs, vintage analog warmth, dancefloor energy, four on the floor",

	"indie rock": "Indie rock with jangly electric guitars, driving rhythm section, melodic bass lines, bright reverbed tones, alternative optimistic energy, mid-tempo",

	"rock": "Classic rock with powerful electric guitar riffs, solid drum grooves, deep bass foundation, raw energetic performance, stadium anthem feeling, driving tempo",
}

// GetCaption returns the ACE-Step generation caption for a genre.
// Returns a generic instrumental caption if the genre has no specific mapping.
func GetCaption(genre string) string {
	if c, ok := captions[genre]; ok {
		return c
	}
	return "Instrumental music, " + genre + " style, professional studio production, warm and immersive sound"
}

// trackWords holds adjective and noun pools for generating track names per genre.
// Names are "adjective noun" pairs like "smoky keys" or "neon grid".
type trackWords struct {
	adjectives []string
	nouns      []string
}

var genreWords = map[string]trackWords{
	"ambient": {
		adjectives: []string{"floating", "weightless", "still", "glacial", "infinite", "deep", "fading", "hollow", "drifting", "crystalline", "submerged", "distant"},
		nouns:      []string{"void", "haze", "tide", "fog", "orbit", "breath", "glacier", "nebula", "silence", "expanse", "shimmer", "glow"},
	},
	"chillwave": {
		adjectives: []string{"hazy", "sunlit", "faded", "dreamy", "pastel", "golden", "washed", "lazy", "coastal", "warm", "blurred", "muted"},
		nouns:      []string{"shore", "sunset", "tape", "glow", "polaroid", "tide", "haze", "mirage", "breeze", "boardwalk", "bloom", "dusk"},
	},
	"lofi hip hop": {
		adjectives: []string{"rainy", "dusty", "warm", "mellow", "quiet", "late", "sleepy", "dimmed", "faded", "cozy", "worn", "gentle"},
		nouns:      []string{"vinyl", "pages", "windowsill", "lamp", "sketch", "rooftop", "notebook", "coffee", "curtain", "alley", "attic", "rain"},
	},
	"jazz": {
		adjectives: []string{"smoky", "midnight", "velvet", "golden", "swinging", "dim", "warm", "slow", "dark", "cool", "muted", "rich"},
		nouns:      []string{"keys", "brass", "lounge", "walk", "quarter", "glass", "satin", "room", "hour", "standard", "corner", "blue"},
	},
	"bossa nova": {
		adjectives: []string{"coastal", "breezy", "gentle", "tropical", "swaying", "soft", "sunlit", "quiet", "warm", "easy", "languid", "tender"},
		nouns:      []string{"terrace", "palm", "wave", "shade", "garden", "hammock", "cove", "balcony", "rain", "veranda", "island", "moon"},
	},
	"acoustic folk": {
		adjectives: []string{"wooded", "fireside", "open", "rustic", "earthen", "amber", "weathered", "honest", "quiet", "still", "humble", "golden"},
		nouns:      []string{"trail", "porch", "valley", "creek", "field", "timber", "lantern", "harvest", "meadow", "cabin", "ridge", "ember"},
	},
	"classical": {
		adjectives: []string{"delicate", "flowing", "stately", "luminous", "grand", "serene", "noble", "graceful", "tender", "poised", "austere", "radiant"},
		nouns:      []string{"sonata", "garden", "waltz", "aria", "hall", "portrait", "reverie", "promenade", "canon", "fountain", "minuet", "passage"},
	},
	"cinematic": {
		adjectives: []string{"epic", "soaring", "vast", "rising", "thundering", "sweeping", "towering", "bold", "triumphant", "distant", "burning", "ancient"},
		nouns:      []string{"horizon", "summit", "empire", "voyage", "fortress", "storm", "dawn", "exodus", "titan", "frontier", "requiem", "ascent"},
	},
	"synthwave": {
		adjectives: []string{"neon", "chrome", "pulsing", "electric", "retro", "violet", "laser", "wired", "digital", "glowing", "turbo", "phantom"},
		nouns:      []string{"grid", "drive", "signal", "circuit", "chase", "vector", "arcade", "skyline", "runner", "highway", "blade", "pulse"},
	},
	"electronic": {
		adjectives: []string{"radiant", "surging", "prismatic", "kinetic", "orbital", "bright", "shifting", "fluid", "charged", "fractal", "rapid", "spectral"},
		nouns:      []string{"pulse", "wave", "prism", "flux", "spark", "cluster", "sphere", "cascade", "node", "beam", "echo", "core"},
	},
	"drum and bass": {
		adjectives: []string{"liquid", "rolling", "dark", "charged", "relentless", "deep", "fractured", "razor", "heavy", "rapid", "volatile", "fierce"},
		nouns:      []string{"drop", "tunnel", "wire", "break", "concrete", "volt", "surge", "pressure", "edge", "shadow", "siren", "vortex"},
	},
	"disco funk": {
		adjectives: []string{"groovy", "sparkling", "tight", "strutting", "vivid", "slick", "golden", "hot", "smooth", "electric", "funky", "glossy"},
		nouns:      []string{"strut", "floor", "mirror", "roller", "glitter", "groove", "spin", "flash", "diva", "velvet", "fever", "step"},
	},
	"indie rock": {
		adjectives: []string{"bright", "jangling", "wistful", "spirited", "raw", "loose", "restless", "hazy", "breezy", "sharp", "earnest", "warm"},
		nouns:      []string{"rooftop", "highway", "postcard", "flicker", "coast", "detour", "signal", "garage", "daylight", "corner", "letter", "echo"},
	},
	"rock": {
		adjectives: []string{"thunderous", "blazing", "driven", "roaring", "massive", "heavy", "fierce", "wild", "burning", "raw", "crushing", "loud"},
		nouns:      []string{"anthem", "riff", "volt", "storm", "iron", "fuel", "hammer", "howl", "forge", "strike", "engine", "thunder"},
	},
}

// kept for backward compat in tests
var genreAdjectives = func() map[string][]string {
	m := make(map[string][]string, len(genreWords))
	for g, w := range genreWords {
		m[g] = w.adjectives
	}
	return m
}()

// trackHash returns a stable unsigned hash from the full track ID string.
func trackHash(s string) uint64 {
	// FNV-1a inspired, uses the entire string for better distribution
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

// TrackName generates a human-readable name from genre and track ID.
// Produces "adjective noun" pairs like "smoky keys" or "neon grid".
func TrackName(genre, trackID string) string {
	if genre == "" || trackID == "" {
		return ""
	}

	words, ok := genreWords[genre]
	if !ok || len(words.adjectives) == 0 {
		return genre + " session"
	}

	h := trackHash(trackID)

	adj := words.adjectives[h%uint64(len(words.adjectives))]
	noun := words.nouns[(h/uint64(len(words.adjectives)))%uint64(len(words.nouns))]

	return adj + " " + noun
}
