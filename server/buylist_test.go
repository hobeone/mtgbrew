package server

import (
	"testing"

	"github.com/hobeone/mtgbrew/mtgjson"
)

func TestSubtractDeck(t *testing.T) {
	c1 := &mtgjson.Card{
		Name: "Foobar",
	}
	c2 := &mtgjson.Card{
		Name: "Baz",
	}

	d1 := DeckList{}
	d1.AddCard(c1, 4)
	d1.AddCard(c2, 4)

	d2 := DeckList{}
	d2.AddCard(c1, 2)
	newd := subtractDeck(d1, d2)
	if newd["Baz"].Count != 4 {
		t.Fatalf("Expected 4 Baz cards got %d", newd["Baz"].Count)
	}
	if newd["Foobar"].Count != 2 {
		t.Fatalf("Expected 2 Foobar cards got %d", newd["Foobar"].Count)
	}
}

func TestTCGList(t *testing.T) {
	c1 := &mtgjson.Card{
		Name: "Foobar",
	}
	c2 := &mtgjson.Card{
		Name: "Baz",
	}

	d1 := DeckList{}
	d1.AddCard(c1, 4)
	d1.AddCard(c2, 4)

	expected := "4 Baz||4 Foobar"
	if d1.TCGList() != expected {
		t.Fatalf("Expected %s got %s ", expected, d1.TCGList())
	}
}

type parseResp struct {
	Name  string
	Count int
	Error error
}

func TestParseLine(t *testing.T) {
	testmap := map[string]parseResp{
		"":                parseResp{"", 0, nil},
		"4 Mountain":      parseResp{"Mountain", 4, nil},
		"4x Mountain":     parseResp{"Mountain", 4, nil},
		"4x Mountain|KLD": parseResp{"Mountain", 4, nil},
		"[sideboard]":     parseResp{"", 0, nil},
		"sideboard":       parseResp{"", 0, nil},
	}

	for k, v := range testmap {
		name, count, err := parseLine(k)
		if name != v.Name {
			t.Errorf("'%s' :: expected Name %s got %s", k, v.Name, name)
		}
		if count != v.Count {
			t.Errorf("'%s' :: expected Count %d got %d", k, v.Count, count)
		}
		if err != v.Error {
			t.Errorf("'%s' :: expected Errors %s got %s", k, v.Error, err)
		}
	}
}
