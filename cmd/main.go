package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhinavxd/artemis/internal/auth"
	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/media"
	"github.com/abhinavxd/artemis/internal/oidc"
	"github.com/abhinavxd/artemis/internal/role"
	"github.com/abhinavxd/artemis/internal/setting"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/template"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko = koanf.New(".")
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	constant     constants
	db           *sqlx.DB
	rdb          *redis.Client
	auth         *auth.Auth
	fs           stuffbin.FileSystem
	i18n         *i18n.I18n
	lo           *logf.Logger
	oidc         *oidc.Manager
	media        *media.Manager
	setting      *setting.Manager
	role         *role.Manager
	contact      *contact.Manager
	user         *user.Manager
	team         *team.Manager
	tag          *tag.Manager
	inbox        *inbox.Manager
	tmpl         *template.Manager
	cannedResp   *cannedresp.Manager
	conversation *conversation.Manager
	automation   *automation.Engine
}

func main() {
	// Load command line flags into Koanf.
	initFlags()

	// Load the config files into Koanf.
	initConfig(ko)

	// Init DB.
	db := initDB()

	fs := initFS()

	// Installer mode? This runs before the SQL queries are loaded and prepared
	// as the installer needs to work on an empty DB.
	if ko.Bool("install") {
		if err := initInstall(db, fs); err != nil {
			log.Fatalf("error running install: %v", err)
		}
		os.Exit(0)
	}

	// Load app settings into Koanf.
	setting := initSettingsManager(db)
	loadSettings(setting)

	var (
		shutdownCh   = make(chan struct{})
		ctx, stop    = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		wsHub        = ws.NewHub()
		i18n         = initI18n(fs)
		lo           = initLogger("artemis")
		rdb          = initRedis()
		oidc         = initOIDC(db)
		auth         = initAuth(oidc, rdb)
		template     = initTemplate(db)
		media        = initMedia(db)
		contact      = initContact(db)
		inbox        = initInbox(db)
		team         = initTeam(db)
		user         = initUser(i18n, db)
		notifier     = initNotifier(user, template)
		automation   = initAutomationEngine(db, user)
		conversation = initConversations(i18n, wsHub, notifier, db, contact, inbox, user, team, media, automation, template)
		autoassigner = initAutoAssigner(team, user, conversation)
	)

	// Set required stores.
	wsHub.SetConversationStore(conversation)
	automation.SetConversationStore(conversation)

	// Register all active inboxes with inbox manager & start receiving messages.
	registerInboxes(inbox, conversation)
	inbox.Receive(ctx)

	// Start evaluation automation rules.
	go automation.Run(ctx)

	// Start conversation auto assigner.
	go autoassigner.Run(ctx, ko.MustDuration("autoassigner.assign_interval"))

	// Listen to incoming messages and dispatch pending outgoing messages.
	go conversation.ListenAndDispatch(ctx, ko.MustInt("message.dispatch_concurrency"), ko.MustInt("message.reader_concurrency"), ko.MustDuration("message.dispatch_read_interval"))

	// CleanMedia deletes media not linked to any model at regular intervals.
	go media.CleanMedia(ctx)

	// Init the app
	var app = &App{
		lo:           lo,
		rdb:          rdb,
		auth:         auth,
		fs:           fs,
		i18n:         i18n,
		media:        media,
		setting:      setting,
		contact:      contact,
		inbox:        inbox,
		user:         user,
		team:         team,
		tmpl:         template,
		conversation: conversation,
		automation:   automation,
		oidc:         oidc,
		role:         initRole(db),
		constant:     initConstants(),
		tag:          initTags(db),
		cannedResp:   initCannedResponse(db),
	}

	// Init fastglue http server.
	g := fastglue.NewGlue()

	// Add app the request context.
	g.SetContext(app)

	// Init the handlers.
	initHandlers(g, wsHub)

	s := &fasthttp.Server{
		Name:                 ko.MustString("app.server.name"),
		ReadTimeout:          ko.MustDuration("app.server.read_timeout"),
		WriteTimeout:         ko.MustDuration("app.server.write_timeout"),
		MaxRequestBodySize:   ko.MustInt("app.server.max_body_size"),
		MaxKeepaliveDuration: ko.MustDuration("app.server.keepalive_timeout"),
		ReadBufferSize:       ko.MustInt("app.server.max_body_size"),
	}

	// Graceful shutdown
	go func() {
		// Wait for the interruption signal
		<-ctx.Done()

		log.Printf("%sShutting down the server. Please wait.\x1b[0m", "\x1b[31m")

		time.Sleep(1 * time.Second)

		// Signal to shutdown the server
		shutdownCh <- struct{}{}
		stop()
	}()

	log.Printf("%s🚀 server listening on %s %s\x1b[0m", "\x1b[32m", ko.String("app.server.address"), ko.String("app.server.socket"))

	if err := g.ListenServeAndWaitGracefully(ko.String("app.server.address"), ko.String("server.socket"), s, shutdownCh); err != nil {
		log.Fatalf("error starting frontend server: %v", err)
	}
}
