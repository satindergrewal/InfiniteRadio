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

// genreAdjectives gives each genre a pool of evocative descriptors for track names.
var genreAdjectives = map[string][]string{
	"ambient":       {"floating", "weightless", "still", "glacial", "infinite"},
	"chillwave":     {"hazy", "sunlit", "faded", "dreamy", "pastel"},
	"lofi hip hop":  {"rainy", "dusty", "warm", "mellow", "quiet"},
	"jazz":          {"smoky", "midnight", "velvet", "golden", "swinging"},
	"bossa nova":    {"coastal", "breezy", "gentle", "tropical", "swaying"},
	"acoustic folk": {"wooded", "fireside", "open", "rustic", "earthen"},
	"classical":     {"delicate", "flowing", "stately", "luminous", "grand"},
	"cinematic":     {"epic", "soaring", "vast", "rising", "thundering"},
	"synthwave":     {"neon", "chrome", "pulsing", "electric", "retro"},
	"electronic":    {"radiant", "surging", "prismatic", "kinetic", "orbital"},
	"drum and bass": {"liquid", "rolling", "dark", "charged", "relentless"},
	"disco funk":    {"groovy", "sparkling", "tight", "strutting", "vivid"},
	"indie rock":    {"bright", "jangling", "wistful", "spirited", "raw"},
	"rock":          {"thunderous", "blazing", "driven", "roaring", "massive"},
}

// TrackName generates a human-readable name from genre and track ID.
// Uses the first 4 hex chars of the ID to pick a deterministic adjective.
func TrackName(genre, trackID string) string {
	if genre == "" || trackID == "" {
		return ""
	}

	adjs := genreAdjectives[genre]
	if len(adjs) == 0 {
		return genre + " session"
	}

	// Use first hex chars of track ID as a simple hash for deterministic pick
	var h int
	for i := 0; i < len(trackID) && i < 8; i++ {
		h = h*31 + int(trackID[i])
	}
	if h < 0 {
		h = -h
	}

	return adjs[h%len(adjs)] + " " + genre
}
