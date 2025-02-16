package utils

import (
	"testing"
)

func TestCeil(t *testing.T) {
	v := Ceil(4.61, 2)

	if float64(4.61) != v {
		t.Errorf("Ceil fail %v: %v", 4.61, v)
	}

	t.Log(v)

	v = Ceil(4.6101, 2)

	if float64(4.62) != v {
		t.Errorf("Ceil fail %v: %v", 4.62, v)
	}

	t.Log(v)

	v = Ceil(4.90910000000001, 1)
	if float64(5) != v {
		t.Errorf("Ceil fail %v: %v", 5, v)
	}

	t.Log(v)

	v = Ceil(4.1111111, 2)
	if float64(4.12) != v {
		t.Errorf("Ceil fail %v: %v", 4.12, v)
	}

	t.Log(v)

	v = Ceil(4.000000, 2)
	if float64(4) != v {
		t.Errorf("Ceil fail %v: %v", 4, v)
	}

	t.Log(v)

	v = Ceil(4, 2)
	if float64(4) != v {
		t.Errorf("Ceil fail %v: %v", 4, v)
	}

	t.Log(v)

	v = Ceil(4.0000000001, 2)
	if float64(4.01) != v {
		t.Errorf("Ceil fail %v: %v", 4.01, v)
	}

	t.Log(v)

	v = Ceil(0.0995, 2)
	if float64(0.1) != v {
		t.Errorf("Ceil fail %v: %v", 4.01, v)
	}

	t.Log(v)

	v = Ceil(0.4090910000000001, 2)
	if float64(0.41) != v {
		t.Errorf("Ceil fail %v: %v", 0.41, v)
	}

	t.Log(v)

	v = Ceil(4.90910000000001, 0)
	if float64(5) != v {
		t.Errorf("Ceil fail %v: %v", 5, v)
	}

	t.Log(v)

	v = Ceil(4.000000001, 0)
	if float64(5) != v {
		t.Errorf("Ceil fail %v: %v", 5, v)
	}

	t.Log(v)
}
