package db

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/hobeone/gomigrate"
	"github.com/hobeone/mtgbrew/mtgjson"
	"github.com/jmoiron/sqlx"
)

var (
	wildcardCols = map[string]bool{
		"name":   true,
		"text":   true,
		"flavor": true,
	}
)

func genSelector(column string, values []string) (string, []string) {
	sel := fmt.Sprintf("%s = ?", column)

	if _, ok := wildcardCols[column]; ok {
		sel = fmt.Sprintf("%s LIKE ?", column)
		for i, v := range values {
			values[i] = "%" + v + "%"
		}
	}

	selectors := make([]string, len(values))
	for i := range values {
		selectors[i] = sel
	}
	final := "(" + strings.Join(selectors, " OR ") + ")"
	return final, values
}

// SearchCards implements advanced searching of the card db
func SearchCards(db *Handle, columns []string, values [][]string) ([]mtgjson.Card, error) {
	selectors, selectvalues := []string{}, []string{}
	for i, col := range columns {
		selector, vals := genSelector(col, values[i])
		selectors = append(selectors, selector)
		selectvalues = append(selectvalues, vals...)
	}
	var interfaceSlice = make([]interface{}, len(selectvalues))
	for i, d := range selectvalues {
		interfaceSlice[i] = d
	}
	queryString := "SELECT * from card WHERE " + strings.Join(selectors, " AND ") + "ORDER BY release_date,name"
	logrus.Infof("query: %s, values: ", queryString, selectvalues)
	cards := []mtgjson.Card{}
	err := db.db.Select(&cards, queryString, interfaceSlice...)
	return cards, err
}

// CardByMTGJsonID returns the first card found with the given mtgjson.com id
func CardByMTGJsonID(dbh *Handle, id string) (*mtgjson.Card, error) {
	card := mtgjson.Card{}
	err := dbh.db.Get(&card, "SELECT * FROM card WHERE mtg_json_id = ? ORDER BY release_date LIMIT 1", id)
	return &card, err
}

// CardByName returns the most recent version of a card with the exact given
// name
func CardByName(dbh *Handle, name string) (*mtgjson.Card, error) {
	card := mtgjson.Card{}
	err := dbh.db.Get(&card, "SELECT * FROM card WHERE search_name = ? ORDER BY release_date DESC LIMIT 1", normalizeName(name))
	return &card, err
}

func normalizeName(name string) string {
	norm := strings.ToLower(name)
	return norm
}

//SaveCards saves all given cards to the db
func SaveCards(db *Handle, sets map[string]mtgjson.Set) error {
	cardInsert := `INSERT INTO card (
"mtg_json_id",
"set_code",
"set_name",
"release_date",
"layout",
"power",
"toughness",
"loyalty",
"hand",
"life",
"cmc",
"mana_cost",
"name",
"names",
"search_name",
"type",
"super_types",
"types",
"sub_types",
"colors",
"rarity",
"text",
"timeshifted",
"reserved",
"starter",
"flavor",
"multiverse_id",
"number",
"source",
"watermark",
"artist",
"image_name") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	for _, set := range sets {
		tx := db.db.MustBegin()
		logrus.Infof("Adding %d cards from %s: %s", len(set.Cards), set.Code, set.Name)
		for _, card := range set.Cards {
			tx.MustExec(cardInsert,
				card.MTGJsonID,
				card.SetCode,
				card.SetName,
				card.ReleaseDate,
				card.Layout,
				card.Power,
				card.Toughness,
				card.Loyalty,
				card.Hand,
				card.Life,
				card.CMC,
				card.ManaCost,
				card.Name,
				card.Names,
				normalizeName(card.Name),
				card.Type,
				card.Supertypes,
				card.Types,
				card.Subtypes,
				card.Colors,
				card.Rarity,
				card.Text,
				card.Timeshifted,
				card.Reserved,
				card.Starter,
				card.Flavor,
				card.MultiverseID,
				card.Number,
				card.Source,
				card.Watermark,
				card.Artist,
				card.ImageName)
		}
		err := tx.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}

// Handle controls access to the database and makes sure only one
// operation is in process at a time.
type Handle struct {
	db        *sqlx.DB
	logger    logrus.FieldLogger
	syncMutex sync.Mutex
}

// NewDBHandle creates a new DBHandle
//	dbPath: the path to the database to use.
//	verbose: when true database accesses are logged to stdout
func NewDBHandle(dbPath string, verbose bool, logger logrus.FieldLogger) *Handle {
	constructedPath := fmt.Sprintf("file:%s?cache=shared&mode=rwc", dbPath)
	db := openDB("sqlite3", constructedPath, verbose, logger)
	err := setupDB(db)
	if err != nil {
		panic(err.Error())
	}
	return &Handle{
		db:     db,
		logger: logger,
	}
}
func openDB(dbType string, dbArgs string, verbose bool, logger logrus.FieldLogger) *sqlx.DB {
	logger.Infof("db: opening database %s:%s", dbType, dbArgs)
	// Error only returns from this if it is an unknown driver.
	db, err := sqlx.Connect("sqlite3", dbArgs)

	if err != nil {
		panic(fmt.Sprintf("Error connecting to %s database %s: %s", dbType, dbArgs, err.Error()))
	}
	// Actually test that we have a working connection
	err = db.Ping()
	if err != nil {
		panic(fmt.Sprintf("db: error connecting to database: %s", err.Error()))
	}
	return db
}

func setupDB(db *sqlx.DB) error {
	_, err := db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA synchronous = NORMAL;")
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA encoding = \"UTF-8\";")
	if err != nil {
		return err
	}

	return nil
}

// NewMemoryDBHandle creates a new in memory database.  Only used for testing.
// The name of the database is a random string so multiple tests can run in
// parallel with their own database.
func NewMemoryDBHandle(verbose bool, logger logrus.FieldLogger, loadFixtures bool) *Handle {
	db := openDB("sqlite3", ":memory:", verbose, logger)

	err := setupDB(db)
	if err != nil {
		panic(err.Error())
	}

	d := &Handle{
		db:     db,
		logger: logger,
	}

	err = d.Migrate(SchemaMigrations())
	if err != nil {
		panic(err)
	}
	/*
		if loadFixtures {
			// load Fixtures
			err = d.Migrate(TestFixtures())
			if err != nil {
				panic(err)
			}
		}
	*/
	return d
}

func migrationsCopy(m []gomigrate.Migration) []*gomigrate.Migration {
	r := make([]*gomigrate.Migration, len(m))
	for i, mig := range m {
		c := mig
		r[i] = &c
	}
	return r
}

// SchemaMigrations gives each caller a new copy of the migrations.  This is
// mostly useful to allow unit tests to run in parallel.
func SchemaMigrations() []*gomigrate.Migration {
	return migrationsCopy(schemaMigrations)
}

// SchemaMigrations contains the series of migrations needed to create and
// update the rss2go db schema.
var schemaMigrations = []gomigrate.Migration{
	{
		ID:   100,
		Name: "Base Schema",
		Up: `CREATE TABLE card (
  "id" INTEGER PRIMARY KEY,
  "mtg_json_id" VARCHAR(255),
  "set_code" VARCHAR(3),
	"set_name" VARCHAR(255),
	"release_date" DATETIME,
  "layout" VARCHAR(255),
  "power" VARCHAR(255),
  "toughness" VARCHAR(255),
  "loyalty" INTEGER,
  "hand" INTEGER,
  "life" INTEGER,
  "cmc" FLOAT,
  "mana_cost" VARCHAR(64),
  "name" VARCHAR(255),
  "names" VARCHAR(255),
	"search_name" VARCHAR(255),
  "type" VARCHAR(255),
  "super_types" VARCHAR(255),
  "types" VARCHAR(255),
  "sub_types" VARCHAR(255),
  "colors" VARCHAR(255),
  "rarity" VARCHAR(255),
  "text" VARCHAR(255),
  "timeshifted" BOOLEAN,
  "reserved" BOOLEAN,
  "starter" BOOLEAN,
  "flavor" TEXT,
  "multiverse_id" INTEGER,
  "number" VARCHAR(255),
  "source" VARCHAR(255),
  "watermark" VARCHAR(255),
  "artist" VARCHAR(255),
  "image_name" VARCHAR(255)
);
CREATE INDEX name_idx on card (name);
CREATE INDEX release_date_name_idx on card (search_name, release_date);
CREATE UNIQUE INDEX mtgjson_idx on card (mtg_json_id)
`,
		Down: `
				"DROP TABLE card",
				`,
	},
}

// Migrate uses the migrations at the given path to update the database.
func (d *Handle) Migrate(m []*gomigrate.Migration) error {
	migrator, err := gomigrate.NewMigratorWithMigrations(d.db.DB, gomigrate.Sqlite3{}, m)
	if err != nil {
		return err
	}
	migrator.Logger = d.logger
	err = migrator.Migrate()
	return err
}
