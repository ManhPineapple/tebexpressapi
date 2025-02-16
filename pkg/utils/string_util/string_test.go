package string_util

import (
	"fmt"
	"testing"
)

func TestA(t *testing.T) {
	itemName := "2tk 035uaap ha if - T-Shirt / L"
	fmt.Println("RemoveInvalidUTF8CharactersAndTrimSpace: ", RemoveInvalidUTF8CharactersAndTrimSpace(itemName))
}

func TestToSlug(t *testing.T) {
	str := "Hà Nội"
	slug := ToSlug(str)

	if slug != "ha-noi" {
		t.Errorf("Convert fail: %s", slug)
	}

	t.Log(slug)
}

func TestToLatin(t *testing.T) {
	str := "ĐỪNG sưa SP_H"
	slug := ToLatinLower(str)
	t.Log(slug)
	if slug != "dung sua sp_h" {
		t.Errorf("Convert fail: %s", slug)
	}

	t.Log(slug)
}
