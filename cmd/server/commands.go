package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
)

var (
	cli struct {
		Globals
		Serve   ServeCmd   `cmd:"serve" help:"start the personal website hosting"`
		Publish PublishCmd `cmd:"publish" help:"uploads some content to the website"`
	}
)

type Globals struct {
	Config string `help:"location of the config path" default:"config.yaml" name:"config.file"`
}

type ServeCmd struct {
	Port  int    `help:"http port for website endpoint (default: 8080)" default:"8080"`
	Host  string `help:"http host for website endpoint (default: localhost)" default:"localhost"`
	Drafts bool   `help:"publish drafts (default: false)" default:"false"`
}

func (c *ServeCmd) Run(globals *Globals) error {
	ctx := context.Background()
	logger := slog.Default()
	// Create the site
	s, _, err := Build(ctx, c.Drafts)
	if err != nil {
		logger.ErrorContext(ctx, "failed to build website", slog.Any("err", err), slog.String("action", "build"))
		os.Exit(1)
	}

	// Create the webmux and attach it to the website
	mux := http.NewServeMux()
	mux.Handle("/", s)

	// Enable logging and serve the website
	logger.InfoContext(ctx, "starting webserver", slog.String("action", "http_listen"))
	if err = http.ListenAndServe(":"+port, mux); err != nil {
		logger.ErrorContext(ctx, "web server shut down unexpectantly", slog.Any("err", err))
	} else {
		logger.InfoContext(ctx, "web server shut down clean")
	}

	return nil
}

type PublishCmd struct {
	Port int      `help:"http port to upload content to (default: 8080)" default:"8080"`
	Host string   `help:"http host to upload content to (default: localhost)" default:"localhost"`
	Post PostCmd  `cmd:"post" help:"upload a post to the website"`
	Note NoteCmd  `cmd:"note" help:"upload a note to the website"`
	Phto PhotoCmd `cmd:"photo" help:"upload a photo to the website"`
}

type PostCmd struct {
	Draft   bool `help:"mark post as draft" default:"false"`
	Publish bool `help:"publish a drafted post" default:"false"`
}

type NoteCmd struct {
	Draft   bool `help:"mark note as draft" default:"false"`
	Publish bool `help:"publish a drafted note" default:"false"`
}

type PhotoCmd struct {
}
