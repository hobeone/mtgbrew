package types

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"

	"github.com/hobeone/mtgbrew/mtgjson"
)

type Card struct {
	ID        uint32 `json:"-"`
	MTGJsonID string `json:"id"`
	Layout    string `json:"layout"`

	Power     string `json:"power,omitempty"`
	Toughness string `json:"toughness,omitempty"`
	Loyalty   int    `json:"loyalty,omitempty"`
	Hand      int    `json:"hand,omitempty"`
	Life      int    `json:"life,omitempty"`

	CMC      float32 `json:"cmc,omitempty"`
	ManaCost string  `json:"manaCost"`

	Name  string   `json:"name"`
	Names []string `json:"names,omitempty"`
	//ForeignNames []ForeignName `json:"foreignNames,omitempty"`
	Type       string   `json:"type"`
	Supertypes []string `json:"supertypes"`
	Types      []string `json:"types"`
	Subtypes   []string `json:"subtypes"`
	Colors     []string `json:"colors"`
	Rarity     string   `json:"rarity"`
	Text       string   `json:"text"`

	Timeshifted bool `json:"timeshifted,omitempty"`
	Reserved    bool `json:"reserved,omitempty"`
	Starter     bool `json:"starter"`

	Flavor string `json:"flavor"`

	MultiverseID int    `json:"multiverseid"` // MULTIVID
	Number       string `json:"number"`
	//	Variations   []int  `json:"variations,omitempty"` // MULTIVID
	Source    string `json:"source,omitempty"`
	Watermark string `json:"watermark,omitempty"`
	Artist    string `json:"artist"`
	ImageName string `json:"imageName"`
	//Legalities   []Legality `json:"legalities"`
	//Rulings      []Ruling   `json:"rulings,omitempty"`
	//	Printings []string `json:"printings"`

	URL      string `json:"url,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	SetURL   string `json:"set_url,omitempty"`
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

func TransformLegalities(lgs []mtgjson.Legality) map[string]string {
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
