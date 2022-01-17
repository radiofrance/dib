package main

import (
	"fmt"
	"os"

	cli "github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"
)

func main() {
	app := cli.App("dib", "Docker Image Builder helps building a complex image dependency graph")

	app.Command("build", "Run docker images builds", cmdBuild)
	app.Command("graph", "Create a visual representation of the build graph", cmdGraph)
	app.Command("test", "Run tests only on docker images. This command expects images references to exist", cmdTest)
	app.Command("hash", "Generate a version hash of the docker directory", cmdHash)
	app.Command("version -v", "Print version information and exit", cmdVersion)

	logLvl := app.String(cli.StringOpt{
		Name:   "l log-level",
		Desc:   "Log level. Can be any level supported by logrus (\"info\", \"debug\", etc...)",
		EnvVar: "LOG_LEVEL",
		Value:  "info",
	})

	app.Before = func() {
		logrusLvl, err := logrus.ParseLevel(*logLvl)
		if err != nil {
			fmt.Printf("Invalid log level %s\n", *logLvl) //nolint:forbidigo

			cli.Exit(1)
		}

		logrus.SetLevel(logrusLvl)
		logrus.SetFormatter(&LogrusTextFormatter{ForceColors: true})
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Errorf("An error occurred: %v", err)
	}
}
