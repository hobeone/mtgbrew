package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/Sirupsen/logrus"
	"github.com/hobeone/mtgbrew/commands"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func setupLogger() {
	fmter := &prefixed.TextFormatter{}
	logrus.SetFormatter(fmter)
	logrus.SetOutput(os.Stdout)
	// Only log the info severity or above.
	logrus.SetLevel(logrus.InfoLevel)
}

var (
	// App is the top level kinping handle
	App    = kingpin.New("mtgbrew", "A Magic The Gathering utility program")
	debug  = App.Flag("debug", "Enable Debug mode.").Bool()
	projID = ""
)

func main() {
	setupLogger()
	commands.RegisterCommands(App)
	kingpin.MustParse(App.Parse(os.Args[1:]))
}
