package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hobeone/mtgbrew/db"
	"github.com/labstack/echo"
)

var (
	// url - column - mapper (=, or like)
	paramMap = map[string]string{
		"name":      "name",
		"cost":      "mana_cost",
		"type":      "types",
		"subtype":   "sub_types",
		"supertype": "super_types",
		"color":     "colors",
		"cmc":       "cmc",
		"power":     "power",
		"toughness": "toughness",
		"text":      "text",
		// To be implemented
		// Multicolor
		"multiverseid": "multiverse_id",
		//"format": "Legalities",
		//"status": "Status"
	}
)

func (a *APIServer) handleCards(c echo.Context) error {
	params := c.QueryParams()
	columns := []string{}
	values := [][]string{}
	for name, column := range paramMap {
		if paramvalue, ok := params[name]; ok {
			columns = append(columns, column)
			values = append(values, lowerStringSlice(paramvalue))
		}
	}

	if len(columns) < 1 {
		return echo.NewHTTPError(http.StatusNoContent, "No arguments given")
	}
	cards, err := db.SearchCards(a.DBH, columns, values)
	if err != nil {
		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	b, err := json.MarshalIndent(cards, "", "  ")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSONBlob(http.StatusOK, b)
}

func (a *APIServer) cardByName(c echo.Context) error {
	cardname := c.Param("name")
	cardname, err := url.QueryUnescape(cardname)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Invalid input: %s", err))
	}
	card, err := db.CardByName(a.DBH, cardname)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No card with that name")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	card.ImageURL = fmt.Sprintf("https://192.168.1.5/img/%s/%s.full.jpg", card.SetCode, card.Name)
	b, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSONBlob(http.StatusOK, b)
}

func (a *APIServer) cardByMyltiverseID(c echo.Context) error {
	cardid := c.Param("id")
	card, err := db.CardByMTGJsonID(a.DBH, cardid)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No card with that id")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	b, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSONBlob(http.StatusOK, b)
}

/*
* Utils
 */

func lowerStringSlice(s []string) []string {
	for i, v := range s {
		s[i] = strings.ToLower(v)
	}
	return s
}
