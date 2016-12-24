package commands

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"github.com/hobeone/mtgbrew/db"
	"github.com/hobeone/mtgbrew/mtgjson"
	// import sqlite driver
	_ "github.com/mattn/go-sqlite3"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// RegisterCommands sets up all the subcommands for Kingpin
func RegisterCommands(app *kingpin.Application) {
	migrate := &migrateSchema{}
	migrate.configure(app)
	load := &loadCardsToDatastore{}
	load.configure(app)
	search := &searchCards{}
	search.configure(app)
	serve := &webServer{}
	serve.configure(app)
}

type migrateSchema struct {
	DBPath string
}

func (m *migrateSchema) configure(app *kingpin.Application) {
	migrate := app.Command("migrate", "crate or migrate schema to current schema").Action(m.Migrate)
	migrate.Flag("dbpath", "Path to database").Required().StringVar(&m.DBPath)
}

func (m *migrateSchema) Migrate(c *kingpin.ParseContext) error {
	dbh := db.NewDBHandle(m.DBPath, true, logrus.StandardLogger())
	return dbh.Migrate(db.SchemaMigrations())
}

type loadCardsToDatastore struct {
	MTGJsonFilePath string
	DBPath          string
}

func (l *loadCardsToDatastore) configure(app *kingpin.Application) {
	loadCards := app.Command("load", "load cards from mtgjson.com to Google Datastore").Action(l.LoadData)
	loadCards.Flag("file", "File containing MTGJson extended set information").Required().StringVar(&l.MTGJsonFilePath)

	loadCards.Flag("dbpath", "Path to database").Required().StringVar(&l.DBPath)

}

func (l *loadCardsToDatastore) LoadData(c *kingpin.ParseContext) error {
	collection, err := mtgjson.LoadCollection(l.MTGJsonFilePath)
	if err != nil {
		return err
	}

	dbh := db.NewDBHandle(l.DBPath, true, logrus.StandardLogger())
	err = db.SaveCards(dbh, collection)

	if err != nil {
		return fmt.Errorf("Error importing cards: %s", err)
	}
	return nil
}

type searchCards struct {
	ProjectID string
	DBPath    string
}

func (s *searchCards) configure(app *kingpin.Application) {
	searchCards := app.Command("search", "search cards").Action(s.Search)

	searchCards.Flag("dbpath", "Path to database").Required().StringVar(&s.DBPath)
}

func (s *searchCards) Search(c *kingpin.ParseContext) error {
	dbh := db.NewDBHandle(s.DBPath, true, logrus.StandardLogger())
	//dbh := NewMemoryDBHandle(true, logrus.StandardLogger(), true)
	cards, err := db.SearchCards(dbh, []string{"name"}, [][]string{[]string{"Selfless Spirit"}})
	if err != nil {
		return err
	}
	spew.Dump(cards)
	return nil
}
