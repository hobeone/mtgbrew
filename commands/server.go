package commands

import (
	"github.com/Sirupsen/logrus"
	"github.com/hobeone/mtgbrew/db"
	"github.com/hobeone/mtgbrew/server"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type webServer struct {
	ProjectID string
	DBPath    string
}

func (s *webServer) configure(app *kingpin.Application) {
	server := app.Command("server", "Start webserver").Action(s.Serve)
	server.Flag("dbpath", "Path to database").Default("mtgcards.db").OverrideDefaultFromEnvar("DBPATH").StringVar(&s.DBPath)
}

func (s *webServer) Serve(c *kingpin.ParseContext) error {
	dbh := db.NewDBHandle(s.DBPath, true, logrus.StandardLogger())

	d := server.Dependencies{
		DBH: dbh,
	}
	server := &server.APIServer{
		Dependencies: d,
	}

	return server.Serve()

}
