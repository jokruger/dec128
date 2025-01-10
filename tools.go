package dec128

// Max returns the largest Dec128 value from the input list.
func Max(a Dec128, b ...Dec128) Dec128 {
	for _, d := range b {
		if d.GreaterThan(a) {
			a = d
		}
	}
	return a
}

// Min returns the smallest Dec128 value from the input list.
func Min(a Dec128, b ...Dec128) Dec128 {
	for _, d := range b {
		if d.LessThan(a) {
			a = d
		}
	}
	return a
}
