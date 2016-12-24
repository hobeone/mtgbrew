package server

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/hobeone/mtgbrew/db"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Dependencies contains all of the things the server needs to run
type Dependencies struct {
	DBH *db.Handle
}

// APIServer implements the API serving part of mtgbrew
type APIServer struct {
	Dependencies
	Port int32
}

// Serve sets up and starts the server
func (s *APIServer) Serve() error {
	e := echo.New()
	e.Debug = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("1024K"))
	e.Use(middleware.Gzip())
	e.Use(headers)

	e.GET("/v1/cards", s.handleCards)
	e.GET("/v1/cardid/:id", s.cardByMyltiverseID)
	e.GET("/v1/card/:name", s.cardByName)

	e.File("/s/buylist", "public/buylist.html")
	e.POST("/v1/buylist", s.formatBuyList)

	e.Static("/img/", "/home/hobe/.forge/pics/cards/")

	t := &Template{
		templates: template.Must(template.New("resp").Parse(`
<!doctype html>

<html lang="en">
<head>
  <meta charset="utf-8">

  <title>MTGBrew Buylist Helper</title>
  <meta name="description" content="mtgbrew buylist">
  <meta name="author" content="mtgbrew">
</head>

<body>
		Errors:
		<ul>
		{{range .Errs}}
		<li>{{.}}</li>
		{{end}}
		</ul>
		<br/>
		DeckList:
		<ul>
		{{range $key, $value := .Deck}}
		<li>{{$value.Count}}  {{$value.Card.Name}}</li>
		{{end}}
		</ul>
		Buy on TCGPlayer <a href="http://store.tcgplayer.com/list/selectproductmagic.aspx?partner=MTGGLDFSH&c={{.BuyLink}}"

</body>
		</html>`)),
	}
	e.Renderer = t

	customServer := &http.Server{
		Addr:           ":7999",
		ReadTimeout:    20 * time.Second,
		WriteTimeout:   20 * time.Second,
		MaxHeaderBytes: 2048,
	}

	err := e.StartServer(customServer)
	if err != nil {
		return fmt.Errorf("Error starting server: %s", err)
	}
	return nil
}

// Template implements the template functionality needed for Echo
type Template struct {
	templates *template.Template
}

// Render implements the echo Render interface
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Set standard headers for all responses
func headers(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//		c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		c.Response().Header().Set("Access-Control-Expose-Headers", "link,content-length")
		c.Response().Header().Set("License", "The textual information presented through this API about Magic: The Gathering is copyrighted by Wizards of the Coast.")
		c.Response().Header().Set("Disclaimer", "This API is not produced, endorsed, supported, or affiliated with Wizards of the Coast.")
		c.Response().Header().Set("Strict-Transport-Security", "max-age=86400")
		return next(c)
	}
}
