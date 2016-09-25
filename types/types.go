package types

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hobeone/mtgformat/gcloud/mtgjson"
)

type Card struct {
	Name          string                `json:"name"`
	ID            string                `json:"id"`
	Href          string                `json:"url,omitempty"`
	StoreURL      string                `json:"store_url"`
	Types         []string              `json:"types,omitempty"`
	Supertypes    []string              `json:"supertypes,omitempty"`
	Subtypes      []string              `json:"subtypes,omitempty"`
	Colors        []string              `json:"colors,omitempty"`
	ConvertedCost int                   `json:"cmc"`
	ManaCost      string                `json:"cost"`
	Text          string                `json:"text"`
	Power         string                `json:"power,omitempty"`
	Toughness     string                `json:"toughness,omitempty"`
	Loyalty       int                   `json:"loyalty,omitempty"`
	Legalities    []mtgjson.MTGLegality `json:"legalities"`
	Editions      []*Edition            `json:"editions,omitempty" datastore:"-"`
}

type Edition struct {
	Set          string `json:"set"`
	SetID        string `json:"set_id"`
	CardID       string `json:"-" datastore:"-"`
	Watermark    string `json:"watermark,omitempty"`
	Rarity       string `json:"rarity"`
	Border       string `json:"-"`
	Artist       string `json:"artist"`
	MultiverseID int    `json:"multiverse_id"`
	Flavor       string `json:"flavor,omitempty"`
	Number       string `json:"number"`
	Layout       string `json:"layout"`
	Href         string `json:"url,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	SetURL       string `json:"set_url,omitempty"`
	StoreURL     string `json:"store_url"`
	HTMLURL      string `json:"html_url"`
}

func (e *Edition) GenKey() string {
	return sha1String(e.Set + "_" + strconv.Itoa(e.MultiverseID))
}

type Set struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Border   string `json:"border"`
	Type     string `json:"type"`
	Href     string `json:"url"`
	CardsURL string `json:"cards_url"`
}

func sha1String(name string) string {
	h := sha1.New()
	h.Write([]byte(name))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func sarray(values []string) string {
	return "{" + strings.Join(values, ",") + "}"
}

func ToSortedLower(things []string) []string {
	sorted := []string{}
	for _, thing := range things {
		sorted = append(sorted, strings.ToLower(strings.Replace(thing, ",", "", -1)))
	}
	sort.Strings(sorted)
	return sorted
}

func transformRarity(rarity string) string {
	r := strings.ToLower(rarity)
	switch r {
	case "mythic rare":
		return "mythic"
	case "basic land":
		return "basic"
	default:
		return r
	}
}

func TransformEdition(s mtgjson.MTGSet, c mtgjson.MTGCard) *Edition {
	return &Edition{
		Set:          s.Name,
		SetID:        s.Code,
		Flavor:       c.Flavor,
		MultiverseID: c.MultiverseId,
		Watermark:    c.Watermark,
		Rarity:       transformRarity(c.Rarity),
		Artist:       c.Artist,
		Border:       c.Border,
		Layout:       c.Layout,
		Number:       c.Number,
		CardID:       sha1String(c.Name),
	}
}

// FIXME: Add released dates
func TransformSet(s mtgjson.MTGSet) Set {
	return Set{
		Name:   s.Name,
		ID:     s.Code,
		Border: s.Border,
		Type:   s.Type,
	}
}

func TransformCard(c mtgjson.MTGCard) Card {
	return Card{
		Name:          c.Name,
		ID:            sha1String(c.Name),
		Text:          c.Text,
		Colors:        ToSortedLower(c.Colors),
		Types:         ToSortedLower(c.Types),
		Supertypes:    ToSortedLower(c.Supertypes),
		Subtypes:      ToSortedLower(c.Subtypes),
		Power:         c.Power,
		Toughness:     c.Toughness,
		Loyalty:       c.Loyalty,
		ManaCost:      c.ManaCost,
		Legalities:    c.Legalities,
		ConvertedCost: int(c.ConvertedCost),
	}
}

func TransformCollection(collection mtgjson.MTGCollection) ([]Set, []Card) {
	cards := []Card{}
	ids := map[string]Card{}
	editions := []*Edition{}
	sets := []Set{}

	for _, set := range collection {
		if strings.HasPrefix(set.Name, "p") {
			continue
		}

		sets = append(sets, TransformSet(set))

		for _, card := range set.Cards {
			newcard := TransformCard(card)
			newedition := TransformEdition(set, card)

			if _, found := ids[newcard.ID]; !found {
				ids[newcard.ID] = newcard
				cards = append(cards, newcard)
			}

			editions = append(editions, newedition)
		}
	}

	for i, c := range cards {
		for _, edition := range editions {
			if edition.CardID == c.ID {
				cards[i].Editions = append(cards[i].Editions, edition)
			}
		}
	}

	return sets, cards
}

func TransformLegalities(lgs []mtgjson.MTGLegality) map[string]string {
	formats := map[string]string{}
	for _, l := range lgs {
		switch l.Format {
		case "Standard":
			formats["standard"] = strings.ToLower(l.Legality)
		case "Modern":
			formats["modern"] = strings.ToLower(l.Legality)
		case "Vintage":
			formats["vintage"] = strings.ToLower(l.Legality)
		case "Legacy":
			formats["legacy"] = strings.ToLower(l.Legality)
		case "Commander":
			formats["commander"] = strings.ToLower(l.Legality)
		}
	}
	return formats
}
