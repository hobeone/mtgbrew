package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/hobeone/mtgbrew/db"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"github.com/labstack/echo/middleware"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type webServer struct {
	ProjectID string
}

func (s *webServer) configure(app *kingpin.Application) {
	app.Command("server", "Start webserver").Action(s.Serve)
}

func (s *webServer) Serve(c *kingpin.ParseContext) error {
	dbh := db.NewDBHandle("foo.db", true, logrus.StandardLogger())

	d := Dependencies{
		DBH: dbh,
	}
	server := &APIServer{
		Dependencies: d,
	}

	e := echo.New()
	e.SetDebug(true)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(headers)

	e.GET("/v1/cards", server.handleCards)
	e.GET("/v1/card/:id", server.cardByMyltiverseID)

	err := e.Run(standard.New(":7999"))
	if err != nil {
		return fmt.Errorf("Error starting server: %s", err)
	}
	return nil
}

// Dependencies contains all of the things the server needs to run
type Dependencies struct {
	DBH *db.Handle
}

// APIServer implements the API serving part of mtgbrew
type APIServer struct {
	Dependencies
}

type searchForm struct {
	Name string   `param:"name"`
	Type []string `param:"type"`
}

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

func lowerStringSlice(s []string) []string {
	for i, v := range s {
		s[i] = strings.ToLower(v)
	}
	return s
}

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

func headers(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		c.Response().Header().Set("Access-Control-Expose-Headers", "link,content-length")
		c.Response().Header().Set("License", "The textual information presented through this API about Magic: The Gathering is copyrighted by Wizards of the Coast.")
		c.Response().Header().Set("Disclaimer", "This API is not produced, endorsed, supported, or affiliated with Wizards of the Coast.")
		c.Response().Header().Set("Pricing", "store.tcgplayer.com allows you to buy cards from any of our vendors, all at the same time, in a simple checkout experience. Shop, Compare & Save with TCGplayer.com!")
		c.Response().Header().Set("Strict-Transport-Security", "max-age=86400")
		return next(c)
	}
}
