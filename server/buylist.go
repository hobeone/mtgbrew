package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hobeone/mtgbrew/db"
	"github.com/hobeone/mtgbrew/mtgjson"
	"github.com/labstack/echo"
)

func readerToDeck(file io.Reader, excludebasic bool, dbh *db.Handle) (DeckList, []error) {
	t := time.Now()
	scanner := bufio.NewScanner(file)
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
		if excludebasic && card.IsBasicLand() {
			continue
		}

		err = deck.AddCard(card, count)
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
				newList.AddCard(cEntry.Card, newCount)
			}
		} else {
			newList.AddCard(entry.Card, entry.Count)
		}
	}
	return newList
}

type formatResp struct {
	Deck DeckList
	Errs []error
}

func (a *APIServer) formatBuyList(c echo.Context) error {
	cardlist := c.FormValue("cardlist")
	subtractcards := c.FormValue("subtractlist")

	excludebasic := false
	if c.FormValue("excludebasic") == "true" {
		excludebasic = true
	}

	var cardreader io.Reader
	if cardlist == "" {
		cardfile, err := c.FormFile("cardlistfile")
		if err != nil {
			return fmt.Errorf("No form or file input given: %s", err)
		}
		src, err := cardfile.Open()
		if err != nil {
			return fmt.Errorf("Error opening form file: %s", err)
		}
		defer src.Close()
		cardreader = src
	} else {
		cardreader = strings.NewReader(cardlist)
	}

	var subtractreader io.Reader
	subtractreader = strings.NewReader(subtractcards)
	if subtractcards == "" { // See if there is a file to read
		cardfile, err := c.FormFile("subtractlistfile")
		if err == nil { // subtract list isn't manditory
			src, err := cardfile.Open()
			if err != nil {
				return fmt.Errorf("Error opening form file: %s", err)
			}
			defer src.Close()
			subtractreader = src
		}
	}
	cards, cardsErrs := readerToDeck(cardreader, excludebasic, a.DBH)
	subcards, subcardsErrs := readerToDeck(subtractreader, excludebasic, a.DBH)

	buylist := subtractDeck(cards, subcards)

	f := formatResp{
		Deck: buylist,
		Errs: append(cardsErrs, subcardsErrs...),
	}
	return c.Render(http.StatusOK, "resp", f)
}

// CardEntry represents the name and count of a particular card in the list
type CardEntry struct {
	Card  *mtgjson.Card
	Count int
}

func (c CardEntry) String() string {
	return fmt.Sprintf("%d %s", c.Count, c.Card.Name)
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
func (d DeckList) AddCard(card *mtgjson.Card, count int) error {
	if card.Name == "" {
		return fmt.Errorf("Card name can't be empty")
	}
	if count < 1 {
		return fmt.Errorf("Card count must be > 0")
	}
	if c, ok := d[card.Name]; ok {
		c.Count = c.Count + count
	} else {
		d[card.Name] = &CardEntry{
			Card:  card,
			Count: count,
		}
	}
	if d[card.Name].Count > 4 && !card.IsBasicLand() {
		d[card.Name].Count = 4
	}

	return nil
}

// TCGList formats Deck for TCGPlayer mass input
// 4 Mountain||4 Forest||...etc
func (d DeckList) TCGList() string {
	l := make([]string, len(d))
	i := 0
	for _, v := range d {
		l[i] = v.String()
		i++
	}
	sort.Strings(l)
	return strings.Join(l, "||")
}

/*
*
* Return empty sring for empty lines and common metadata lines
* eg: sideboard etc.
*
 */
func parseLine(line string) (string, int, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", 0, nil
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
