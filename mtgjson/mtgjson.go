package mtgjson

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
)

// StringSlice implements the Valuer/Scanner interfaces to save to the database
type StringSlice []string

// Value implements the Valuer interface for SQL operations
func (s StringSlice) Value() (driver.Value, error) {
	return strings.Join([]string(s), ","), nil
}

// Scan implements the Scanner interface for database/sql
func (s *StringSlice) Scan(src interface{}) error {
	var source string
	switch src.(type) {
	case string:
		source = src.(string)
	case []byte:
		source = string(src.([]byte))
	default:
		return errors.New("Incpompatible type for StringSlice")
	}
	*s = StringSlice(strings.Split(source, ","))
	return nil
}

// ToLower returns a new StringSlice with all elemets lowercased
func (s StringSlice) ToLower() StringSlice {
	for i, v := range s {
		s[i] = strings.ToLower(v)
	}
	return s
}

// Set represents a particular set of magic cards
type Set struct {
	Name               string  `json:"name"`
	Code               string  `json:"code"`
	GathererCode       string  `json:"gathererCode,omitempty"`
	OldCode            string  `json:"oldCode,omitempty"`
	MagicCardsInfoCode string  `json:"magicCardsInfoCode,omitempty"`
	ReleaseDate        string  `json:"releaseDate"`
	Border             string  `json:"border"`
	SetType            string  `json:"type"`
	Block              string  `json:"block"`
	OnlineOnly         bool    `json:"onlineOnly,omitempty"`
	Cards              []*Card `json:"cards"`
}

// Card represents a Magic Card from a particular set
type Card struct {
	ID        uint32 `json:"-"`
	MTGJsonID string `json:"id" db:"mtg_json_id"`
	SetCode   string `json:"set" db:"set_code"`
	SetName   string `json:"set" db:"set_name"`
	Layout    string `json:"layout"`

	Power     string `json:"power,omitempty"`
	Toughness string `json:"toughness,omitempty"`
	Loyalty   int    `json:"loyalty,omitempty"`
	Hand      int    `json:"hand,omitempty"`
	Life      int    `json:"life,omitempty"`

	CMC      float32 `json:"cmc,omitempty"`
	ManaCost string  `json:"manaCost" db:"mana_cost"`

	Name  string      `json:"name"`
	Names StringSlice `json:"names,omitempty"`
	//ForeignNames []ForeignName `json:"foreignNames,omitempty"`
	Type       string      `json:"type"`
	Supertypes StringSlice `json:"supertypes" db:"super_types"`
	Types      StringSlice `json:"types"`
	Subtypes   StringSlice `json:"subtypes" db:"sub_types"`
	Colors     StringSlice `json:"colors"`
	Rarity     string      `json:"rarity"`
	Text       string      `json:"text"`

	Timeshifted bool `json:"timeshifted,omitempty"`
	Reserved    bool `json:"reserved,omitempty"`
	Starter     bool `json:"starter"`

	Flavor string `json:"flavor"`

	MultiverseID int    `json:"multiverseid" db:"multiverse_id"`
	Number       string `json:"number"`
	//	Variations   []int  `json:"variations,omitempty"` // MULTIVID
	Source    string `json:"source,omitempty"`
	Watermark string `json:"watermark,omitempty"`
	Artist    string `json:"artist"`
	ImageName string `json:"imageName" db:"image_name"`
	//Legalities   []Legality `json:"legalities"`
	//Rulings      []Ruling   `json:"rulings,omitempty"`
	//	Printings []string `json:"printings"`

	URL      string `json:"url,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	SetURL   string `json:"set_url,omitempty"`
}

type Legality struct {
	Format   string `json:"format"`
	Legality string `json:"legality"`
	//Condition string `json:"condition,omitempty"`
}

type Ruling struct {
	Date string `json:"date"`
	Text string `json:"text"`
}

type ForeignName struct {
	Lang         string `json:"language"`
	Name         string `json:"name"`
	MultiverseID int    `json:"multiverseid"` // MULTIVID
}

func processCard(card *Card) error {
	card.SetCode = strings.ToLower(card.SetCode)
	card.SetName = strings.ToLower(card.SetName)
	card.Layout = strings.ToLower(card.Layout)
	card.Power = strings.ToLower(card.Power)
	card.Toughness = strings.ToLower(card.Toughness)
	card.Name = strings.ToLower(card.Name)
	card.Names = card.Names.ToLower()
	card.Type = strings.ToLower(card.Type)
	card.Supertypes = card.Supertypes.ToLower()
	card.Types = card.Types.ToLower()
	card.Subtypes = card.Subtypes.ToLower()
	card.Colors = card.Colors.ToLower()
	card.Rarity = strings.ToLower(card.Rarity)
	card.Text = strings.ToLower(card.Text)
	card.Flavor = strings.ToLower(card.Flavor)
	card.Number = strings.ToLower(card.Number)
	card.Source = strings.ToLower(card.Source)
	card.Watermark = strings.ToLower(card.Watermark)
	card.Artist = strings.ToLower(card.Artist)
	card.ImageName = strings.ToLower(card.ImageName)
	return nil
}

// LoadCollection unmarshals a mtgjson.com data dump into Set & Card structs
func LoadCollection(path string) (map[string]Set, error) {
	blob, err := ioutil.ReadFile(path)
	setmap := make(map[string]Set)

	if err != nil {
		return setmap, err
	}

	err = json.Unmarshal(blob, &setmap)
	if err != nil {
		return nil, err
	}
	for _, set := range setmap {
		for _, card := range set.Cards {
			card.SetCode = set.Code
			card.SetName = set.Name
			processCard(card)
		}
	}
	return setmap, err
}
