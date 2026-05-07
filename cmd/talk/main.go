package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/fbegyn/website/internal/auth"
	"github.com/fbegyn/website/internal/blog"
	"github.com/fbegyn/website/internal/contextkey"
	"github.com/fbegyn/website/internal/middleware"
	"github.com/fbegyn/website/internal/multiplex"
	"github.com/fbegyn/website/internal/talkrender"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"within.website/ln/ex"
)

var (
	talkViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "talk_views",
			Help: "number of views per talk",
		}, []string{"talk"},
	)
	pageViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "page_views",
			Help: "number of views for a page",
		}, []string{"page"},
	)
)

var cli struct {
	Globals
	Serve ServeCmd `cmd:"serve" help:"start the talk server"`
}

type Globals struct {
	Config string `help:"location of the config path" default:"config.yaml" name:"config.file"`
}

type ServeCmd struct {
	Port          string `help:"http port for talk server endpoint (default: 3115)" default:"3115"`
	Host          string `help:"http host for talk server endpoint (default: localhost)" default:"localhost"`
	Drafts        bool   `help:"publish drafts (default: false)" default:"false"`
	TalksDir      string `help:"directory containing talk presentations" default:"./talks/"`
	StaticDir     string `help:"directory containing static files" default:"./static"`
	PresenterUser string `help:"basic-auth username gating /talks/presenter/* (empty = disable)" env:"TALK_PRESENTER_USER" default:""`
	PresenterPass string `help:"basic-auth password gating /talks/presenter/* (empty = disable)" env:"TALK_PRESENTER_PASS" default:""`
}

func (c *ServeCmd) Run(globals *Globals) error {
	ctx := context.Background()
	logger := slog.Default()

	// Create the talk server
	s, err := BuildTalkServer(ctx, c.TalksDir, c.StaticDir, c.Drafts, c.PresenterUser, c.PresenterPass)
	if err != nil {
		logger.ErrorContext(ctx, "failed to build talk server", slog.Any("err", err), slog.String("action", "build"))
		os.Exit(1)
	}

	// Create the webmux and attach it to the talk server. Multiplex
	// routes (token / SSE / WS) sit on the outer mux so they skip
	// ex.HTTPLog, whose response wrapper blocks the WebSocket Hijack.
	mux := http.NewServeMux()
	s.Multiplex().Register(mux)
	mux.Handle("/", s)

	// Enable logging and serve the website
	logger.InfoContext(ctx, "starting talk server", slog.String("action", "http_listen"), slog.String("port", c.Port))
	if err = http.ListenAndServe(":"+c.Port, mux); err != nil {
		logger.ErrorContext(ctx, "talk server shut down unexpectantly", slog.Any("err", err))
	} else {
		logger.InfoContext(ctx, "talk server shut down clean")
	}

	return nil
}

// TalkServer represents the talk server structure and data
type TalkServer struct {
	Talks     blog.Talks
	Drafts    bool
	TalksDir  string
	StaticDir string

	mux       *http.ServeMux
	xffmw     *xff.XFF
	multiplex *multiplex.Hub
}

// Multiplex exposes the in-process multiplex Hub so the outer mux can
// mount its routes ahead of the logging chain.
func (s *TalkServer) Multiplex() *multiplex.Hub { return s.multiplex }

// ServeHTTP makes TalkServer implement http.Handler
func (s *TalkServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextkey.Key("func"), "talkserver.ServeHTTP")
	ctx = context.WithValue(ctx, contextkey.Key("user_agent"), r.Header.Get("User-Agent"))
	r = r.WithContext(ctx)
	middleware.RequestID(s.xffmw.Handler(ex.HTTPLog(s.mux))).ServeHTTP(w, r)
}

// BuildTalkServer creates and configures the talk server. presenterUser
// and presenterPass guard the /talks/presenter routes; if both are
// empty the routes are not registered.
func BuildTalkServer(ctx context.Context, talksDir, staticDir string, publishDrafts bool, presenterUser, presenterPass string) (*TalkServer, error) {
	ctx = context.WithValue(ctx, contextkey.Key("func"), "BuildTalkServer")

	// Handle X-Forwarded-For headers
	xffmw, err := xff.Default()
	if err != nil {
		return nil, err
	}

	// Struct that represents the talk server
	s := &TalkServer{
		mux:       http.NewServeMux(),
		xffmw:     xffmw,
		TalksDir:  talksDir,
		StaticDir: staticDir,
		multiplex: multiplex.New(),
	}

	// Load talk entries from disk
	talks, err := blog.LoadTalksDir(talksDir, "talk", publishDrafts)
	if err != nil {
		return nil, err
	}
	s.Talks = talks

	// Add HTTP routes
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			s.renderPageTemplate("error.html", "page not found: "+r.URL.Path).ServeHTTP(w, r)
			return
		}
		// Redirect to /talks
		http.Redirect(w, r, "/talks", http.StatusSeeOther)
	})

	s.mux.Handle("GET /metrics", promhttp.Handler())
	s.mux.Handle("GET /talks", middleware.Metrics("talk", s.renderPageTemplate("talks/overview.html", s.Talks)))
	s.mux.Handle("GET /talks/{year}/{slug}", middleware.Metrics("talks", http.HandlerFunc(s.renderTalk)))
	s.mux.Handle("GET /talks/{year}/{slug}/{socketID}", middleware.Metrics("talks", http.HandlerFunc(s.renderTalk)))

	// Serve static files first (more general handler)
	s.mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServer(http.Dir(s.StaticDir))))

	// Handle talk content files (this is for serving the talk markdown/HTML files)
	t := blog.TalkFS{BaseDir: talksDir}
	s.mux.Handle("GET /static/talks/", http.StripPrefix(
		"/static/",
		blog.TalkFSHandler(http.FileServerFS(t)),
	))
	s.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(s.StaticDir, "favicon.ico"))
	})

	// Multiplex routes are mounted on the *outer* mux by ServeCmd.Run
	// so they bypass within.website/ln/ex.HTTPLog — that logger wraps
	// the response writer without exposing http.Hijacker, which the WS
	// upgrade needs. See TalkServer.Multiplex().

	if presenterUser != "" && presenterPass != "" {
		s.mux.Handle("GET /talks/presenter/{year}/{slug}", middleware.Metrics("talks", http.HandlerFunc(
			auth.Basic(presenterUser, presenterPass, middleware.MultiplexCreateCredentials(s.multiplex, s.renderTalk)),
		)))
		s.mux.Handle("GET /talks/presenter/{year}/{slug}/{socketID}/{secret}", middleware.Metrics("talks", http.HandlerFunc(
			auth.Basic(presenterUser, presenterPass, middleware.MultiplexCreateCredentials(s.multiplex, s.renderTalk)),
		)))
		slog.Info("presenter control available at /talks/presenter/...")
	} else {
		slog.Info("presenter control disabled (set --presenter-user/--presenter-pass to enable)")
	}

	return s, nil
}

func (s *TalkServer) renderPageTemplate(templateFile string, data interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var t *template.Template
		var err error
		t, err = template.ParseFiles("templates/base.html", "templates/"+templateFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error(
				"failed to render page",
				"error", err,
				"action", "renderPageTemplate",
				"page", templateFile,
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Cache-Control", "max-age=86400")

		err = t.Execute(w, data)
		if err != nil {
			slog.Error(
				"failed to execute template",
				"error", err,
				"action", "executeTemplate",
				"page", templateFile,
			)
		}
		pageViews.With(prometheus.Labels{"page": templateFile}).Inc()
	})
}

func (s *TalkServer) renderTalk(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "talks/" {
		http.Redirect(w, r, "/talks", http.StatusSeeOther)
		return
	}
	p, ok := talkrender.Find(s.Talks, r.PathValue("year"), r.PathValue("slug"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		s.renderPageTemplate("error.html", "no such talk found: "+r.RequestURI).ServeHTTP(w, r)
		return
	}
	talkrender.Render(w, r, p)
	talkViews.With(prometheus.Labels{"talk": filepath.Base(p.Slug)}).Inc()
}

func main() {
	appCtx := kong.Parse(&cli,
		kong.Name("talk-server"),
		kong.Description("Standalone talk/presentation server"),
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
