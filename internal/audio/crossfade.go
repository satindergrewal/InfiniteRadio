package audio

// Smoothstep returns the smoothstep interpolation for t in [0,1].
// Formula: 3t^2 - 2t^3 (same curve as original InfiniteRadio Python code).
func Smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// CrossfadeFrames blends an outgoing frame with an incoming frame at the given
// progress (0.0 = all outgoing, 1.0 = all incoming). Uses smoothstep curve.
// Both frames must have the same length. Returns the blended frame.
func CrossfadeFrames(outgoing, incoming []int16, progress float64) []int16 {
	gain := Smoothstep(progress)
	result := make([]int16, len(outgoing))

	for i := range outgoing {
		out := float64(outgoing[i]) * (1 - gain)
		in := float64(incoming[i]) * gain
		mixed := out + in

		// Clip to int16 range
		if mixed > 32767 {
			mixed = 32767
		} else if mixed < -32768 {
			mixed = -32768
		}
		result[i] = int16(mixed)
	}

	return result
}
