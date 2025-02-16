package packages

const GramToPound = 0.00220462262185
const MaxDimension = 270
const MaxLength = 120
const MinLength = 120

func parseVolumes(d, r, c float64) (length, width, height float64) {
	width = r
	length = d
	height = c

	if length < height {
		height, length = length, height
	}

	if width < height {
		height, width = width, height
	}

	if length < width {
		width, length = length, width
	}

	return
}

func IsBigsize(weight, length, width, height float64) bool {
	d, r, c := parseVolumes(length, width, height)
	if d <= MinLength {
		return false
	}

	if d > 120 {
		return false
	}

	dv := d + 2*(r+c)
	if dv > 270 {
		return false
	}

	if weight >= 42 {
		return false
	}

	return true
}
