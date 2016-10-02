package mtgjson

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestLoadCardJSON(t *testing.T) {
	collection, err := LoadCollection("testsets.json")

	if err != nil {
		t.Fatal(err)
	}

	set, ok := collection["LEA"]

	if !ok {
		t.Fatal("The collection did not load properly")
	}
	spew.Dump(set.ReleaseDate)

	if len(set.Cards) != 1 {
		t.Fatalf("Expected 1 card got %d", len(set.Cards))
	}
	spew.Dump(set.Cards)
}
