package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/alecthomas/kong"
)

var (
	port    = os.Getenv("SERVER_PORT")
	arbDate = time.Date(2020, time.January, 9, 0, 0, 0, 0, time.UTC)
)

func main() {
	// Port selection for the webste
	if port == "" {
		port = "3114"
	}

	// create a context in which the website can run and add logging
	// ctx := context.Background()
	// logger := slog.Default()
	// slog.SetDefault(logger)
	// logger = slog.Default().With(slog.String("func", "main"))

	appCtx := kong.Parse(&cli,
		kong.Name("fb-website"),
		kong.Description("Personal website hosting and CLI tooling"),
		kong.UsageOnError(),
		kong.Vars{
			"version": "0.1.0",
		},
	)

	err := appCtx.Run(&cli.Globals)
	if err != nil {
		slog.Error("failed to run kong app", "error", err)
		os.Exit(5)
	}
}
