package server

import (
	"bufio"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hobeone/mtgbrew/db"
	"github.com/labstack/echo"
)

func stringToDeck(cards string, excludebasic bool, dbh *db.Handle) (DeckList, []error) {
	t := time.Now()
	cardreader := strings.NewReader(cards)
	scanner := bufio.NewScanner(cardreader)
	deck := DeckList{}
	errs := []error{}
	linecount := 0

	for scanner.Scan() {
		name, count, err := parseLine(scanner.Text())
		linecount++
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if name == "" {
			continue
		}
		card, err := db.CardByName(dbh, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("Unknown card: '%s' (%s)", name, err))
			continue
		}
		if excludebasic && card.Rarity == "Basic Land" {
			continue
		}

		err = deck.AddCard(card.Name, count)
		if err != nil {
			errs = append(errs, err)
		}
	}
	logrus.Infof("Took %v seconds to process %d lines", time.Now().Sub(t), linecount)
	return deck, errs
}

func subtractDeck(newDeck, collection DeckList) DeckList {
	newList := DeckList{}
	for name, entry := range newDeck {
		if cEntry, ok := collection[name]; ok {
			newCount := entry.Count - cEntry.Count
			if newCount > 0 {
				newList.AddCard(name, newCount)
			}
		} else {
			newList.AddCard(name, entry.Count)
		}
	}
	return newList
}

type formatResp struct {
	Deck DeckList
	Errs []error
}

func (a *APIServer) formatBuyList(c echo.Context) error {
	//TODO: input length protection
	name := c.FormValue("cardlist")
	excludebasic := false
	if c.FormValue("excludebasic") == "true" {
		excludebasic = true
	}
	subtractcards := c.FormValue("subtractlist")

	cards, cardsErrs := stringToDeck(name, excludebasic, a.DBH)
	subcards, subcardsErrs := stringToDeck(subtractcards, excludebasic, a.DBH)

	buylist := subtractDeck(cards, subcards)

	return c.Render(http.StatusOK, "resp", formatResp{buylist, append(cardsErrs, subcardsErrs...)})
}

// CardEntry represents the name and count of a particular card in the list
type CardEntry struct {
	Name  string
	Count int
}

func (c CardEntry) String() string {
	return fmt.Sprintf("%d %s", c.Count, c.Name)
}

// DeckList represents a set of cards
type DeckList map[string]*CardEntry

func (d DeckList) String() string {
	keys := make([]string, len(d))
	i := 0
	for k := range d {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	retStrs := make([]string, len(keys))
	for i, k := range keys {
		retStrs[i] = d[k].String()
	}
	return strings.Join(retStrs, "\n")
}

// AddCard adds a card to the deck up to a max of 4 except for basic lands
func (d DeckList) AddCard(name string, count int) error {
	if name == "" {
		return fmt.Errorf("Card name can't be empty")
	}
	if count < 1 {
		return fmt.Errorf("Card count must be > 0")
	}
	if c, ok := d[name]; ok {
		c.Count = c.Count + count
	} else {
		d[name] = &CardEntry{
			Name:  name,
			Count: count,
		}
	}
	if d[name].Count > 4 {
		d[name].Count = 4
	}

	return nil
}

func parseLine(line string) (string, int, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", 0, nil // Empty line error?
	}
	matched, err := regexp.MatchString(`(?i)^\[?side`, line)
	if matched {
		return "", 0, nil
	}
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("Bad line format: '%s'", line)
	}
	nameparts := strings.SplitN(parts[1], "/", 2)
	name := nameparts[0]
	nameparts = strings.SplitN(name, "|", 2) // dck format includes set: Abbot of Keral Keep|ORI
	name = nameparts[0]
	parts[0] = strings.TrimRight(parts[0], "x") // For 1x Mountain
	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", 0, fmt.Errorf("Invalid Count: '%s'", parts[0])
	}
	return name, count, nil
}
