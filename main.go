package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/Sirupsen/logrus"
	"github.com/hobeone/mtgformat/gcloud/mtgjson"
	"github.com/hobeone/mtgformat/gcloud/types"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"cloud.google.com/go/datastore"
)

func saveCards(client *datastore.Client, path string) error {
	collection, err := mtgjson.LoadCollection(path)
	if err != nil {
		return err
	}
	_, cards := types.TransformCollection(collection)

	ctx := context.Background()
	cardKeys := make([]*datastore.Key, len(cards))
	editionKeys := [][]*datastore.Key{}
	editions := [][]*types.Edition{}
	for i, card := range cards {
		key := datastore.NewKey(ctx, "MTGCard", card.ID, 0, nil)
		cardKeys[i] = key
		edkeys := make([]*datastore.Key, len(card.Editions))
		for k, ed := range card.Editions {
			edkeys[k] = datastore.NewKey(ctx, "Edition", ed.GenKey(), 0, key)
		}
		editionKeys = append(editionKeys, edkeys)
		editions = append(editions, card.Editions)
	}

	start, end := 0, 500
	for start < len(cards) {
		if end > len(cards) {
			end = len(cards)
		}
		logrus.Infof("Saving cards %d to %d of %d\n", start, end, len(cards))
		_, err := client.PutMulti(ctx, cardKeys[start:end], cards[start:end])
		if err != nil {
			return fmt.Errorf("Error saving cards: %s", err)
		}
		start = end + 1
		end = start + 500
	}
	for i, eds := range editions {
		_, err := client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			logrus.Infof("Saving Editions for card %d of %d\n", i, len(editions))
			_, err := tx.PutMulti(editionKeys[i], eds)
			if err != nil {
				return fmt.Errorf("Error saving Editions: %s", err)
			}
			return nil
		}, datastore.MaxAttempts(2))
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	// App is the top level kinping handle
	App    = kingpin.New("mtgbrew", "A Magic The Gathering utility program")
	debug  = App.Flag("debug", "Enable Debug mode.").Bool()
	projID = ""
)

type loadCardsToDatastore struct {
	MTGJsonFilePath string
}

func (l *loadCardsToDatastore) configure(app *kingpin.Application) {
	loadCards := app.Command("load", "load cards from mtgjson.com to Google Datastore").Action(l.LoadData)
	loadCards.Flag("file", "File containing MTGJson extended set information").Required().StringVar(&l.MTGJsonFilePath)
}

func (l *loadCardsToDatastore) LoadData(c *kingpin.ParseContext) error {
	// [START build_service]
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projID)
	// [END build_service]
	if err != nil {
		return fmt.Errorf("Could not create datastore client: %v", err)
	}

	err = saveCards(client, l.MTGJsonFilePath)
	if err != nil {
		return fmt.Errorf("Error importing cards: %s", err)
	}
	return nil
}

type searchCards struct{}

func (s *searchCards) configure(app *kingpin.Application) {
	app.Command("search", "search cards").Action(s.Search)
}

func (s *searchCards) Search(c *kingpin.ParseContext) error {
	// [START build_service]
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projID)
	if err != nil {
		return fmt.Errorf("Could not create datastore client: %v", err)
	}

	var cards []*types.Card

	queries := []*datastore.Query{}
	//queries = append(queries, datastore.NewQuery("MTGCard").Filter("Name =", "Mountain"))
	//queries = append(queries, datastore.NewQuery("MTGCard").Filter("Name =", "Serra Angel"))
	queries = append(queries, datastore.NewQuery("MTGCard").Filter("Name =", "Tarmogoyf"))
	/*	queries = append(queries, datastore.NewQuery("MTGCard").Filter("Types =", "Instant").Filter("ManaCost =", "{B}{B}"))
		queries = append(queries, datastore.NewQuery("MTGCard").Filter("Colors =", "Black").Filter("ManaCost =", "{B}{B}"))
		queries = append(queries, datastore.NewQuery("MTGCard").Filter("Colors =", "Black").Filter("ManaCost =", "{B}{B}").Filter("Types =", "Instant"))
	*/
	for _, q := range queries {
		cards = []*types.Card{}
		keys, err := client.GetAll(ctx, q, &cards)
		if err != nil {
			return err
		}
		for i, key := range keys {
			cards[i].ID = key.Name()
		}
	}
	for _, card := range cards {
		key := datastore.NewKey(ctx, "MTGCard", card.ID, 0, nil)
		q := datastore.NewQuery("Edition").Ancestor(key)
		var eds []*types.Edition
		_, err := client.GetAll(ctx, q, &eds)
		if err != nil {
			return err
		}
		card.Editions = eds
		blob, err := json.MarshalIndent(card, "", "  ")
		fmt.Println(string(blob))
		fmt.Printf("%s - %s - %s\n", card.Name, card.Types, card.ManaCost)
	}
	return nil

}

func setupLogger() {
	fmter := &prefixed.TextFormatter{}
	logrus.SetFormatter(fmter)
	logrus.SetOutput(os.Stdout)
	// Only log the info severity or above.
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	setupLogger()

	projID := os.Getenv("DATASTORE_PROJECT_ID")
	if projID == "" {
		logrus.Fatal(`You need to set the environment variable "DATASTORE_PROJECT_ID"`)
	}

	load := &loadCardsToDatastore{}
	load.configure(App)
	search := &searchCards{}
	search.configure(App)
	kingpin.MustParse(App.Parse(os.Args[1:]))

}
