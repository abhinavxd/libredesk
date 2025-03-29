package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/abhinavxd/libredesk/internal/ai"
	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	"github.com/abhinavxd/libredesk/internal/authz"
	businesshours "github.com/abhinavxd/libredesk/internal/business_hours"
	"github.com/abhinavxd/libredesk/internal/colorlog"
	"github.com/abhinavxd/libredesk/internal/csat"
	"github.com/abhinavxd/libredesk/internal/macro"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/search"
	"github.com/abhinavxd/libredesk/internal/sla"
	"github.com/abhinavxd/libredesk/internal/view"

	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/conversation/priority"
	"github.com/abhinavxd/libredesk/internal/conversation/status"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/oidc"
	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/tag"
	"github.com/abhinavxd/libredesk/internal/team"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/knadh/go-i18n"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko          = koanf.New(".")
	ctx         = context.Background()
	appName     = "libredesk"
	frontendDir = "frontend/dist"

	// Injected at build time.
	buildString   string
	versionString string
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	fs            stuffbin.FileSystem
	consts        atomic.Value
	auth          *auth_.Auth
	authz         *authz.Enforcer
	i18n          *i18n.I18n
	lo            *logf.Logger
	oidc          *oidc.Manager
	media         *media.Manager
	setting       *setting.Manager
	role          *role.Manager
	user          *user.Manager
	team          *team.Manager
	status        *status.Manager
	priority      *priority.Manager
	tag           *tag.Manager
	inbox         *inbox.Manager
	tmpl          *template.Manager
	macro         *macro.Manager
	conversation  *conversation.Manager
	automation    *automation.Engine
	businessHours *businesshours.Manager
	sla           *sla.Manager
	csat          *csat.Manager
	view          *view.Manager
	ai            *ai.Manager
	search        *search.Manager
	notifier      *notifier.Service

	// Global state that stores data on an available app update.
	update *AppUpdate
	sync.Mutex
}

func main() {
	// Set up signal handler.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load command line flags into Koanf.
	initFlags()

	// Version flag.
	if ko.Bool("version") {
		fmt.Println(buildString)
		os.Exit(0)
	}

	// Build string injected at build time.
	colorlog.Green("Build: %s", buildString)

	// Load the config files into Koanf.
	initConfig(ko)

	// Init stuffbin fs.
	fs := initFS()

	// Init DB.
	db := initDB()

	// Installer.
	if ko.Bool("install") {
		install(ctx, db, fs, ko.Bool("idempotent-install"), !ko.Bool("yes"))
		os.Exit(0)
	}

	// Set system user password.
	if ko.Bool("set-system-user-password") {
		setSystemUserPass(ctx, db)
		os.Exit(0)
	}

	// Check if schema is installed.
	installed, err := checkSchema(db)
	if err != nil {
		log.Fatalf("error checking db schema: %v", err)
	}
	if !installed {
		log.Println("database tables are missing. Use the `--install` flag to set up the database schema.")
		os.Exit(0)
	}

	// Upgrade.
	if ko.Bool("upgrade") {
		upgrade(db, fs, !ko.Bool("yes"))
		os.Exit(0)
	}

	// Check for pending upgrade.
	checkPendingUpgrade(db)

	// Load app settings from DB into the Koanf instance.
	settings := initSettings(db)
	loadSettings(settings)

	var (
		autoAssignInterval          = ko.MustDuration("autoassigner.autoassign_interval")
		unsnoozeInterval            = ko.MustDuration("conversation.unsnooze_interval")
		automationWorkers           = ko.MustInt("automation.worker_count")
		messageOutgoingQWorkers     = ko.MustDuration("message.outgoing_queue_workers")
		messageIncomingQWorkers     = ko.MustDuration("message.incoming_queue_workers")
		messageOutgoingScanInterval = ko.MustDuration("message.message_outoing_scan_interval")
		slaEvaluationInterval       = ko.MustDuration("sla.evaluation_interval")
		lo                          = initLogger(appName)
		rdb                         = initRedis()
		constants                   = initConstants()
		i18n                        = initI18n(fs)
		csat                        = initCSAT(db, i18n)
		oidc                        = initOIDC(db, settings, i18n)
		status                      = initStatus(db, i18n)
		priority                    = initPriority(db, i18n)
		auth                        = initAuth(oidc, rdb, i18n)
		template                    = initTemplate(db, fs, constants, i18n)
		media                       = initMedia(db, i18n)
		inbox                       = initInbox(db, i18n)
		team                        = initTeam(db, i18n)
		businessHours               = initBusinessHours(db, i18n)
		user                        = initUser(i18n, db)
		wsHub                       = initWS(user)
		notifier                    = initNotifier()
		automation                  = initAutomationEngine(db, i18n)
		sla                         = initSLA(db, team, settings, businessHours, notifier, template, user, i18n)
		conversation                = initConversations(i18n, sla, status, priority, wsHub, notifier, db, inbox, user, team, media, settings, csat, automation, template)
		autoassigner                = initAutoAssigner(team, user, conversation)
	)
	automation.SetConversationStore(conversation)

	startInboxes(ctx, inbox, conversation)
	go automation.Run(ctx, automationWorkers)
	go autoassigner.Run(ctx, autoAssignInterval)
	go conversation.Run(ctx, messageIncomingQWorkers, messageOutgoingQWorkers, messageOutgoingScanInterval)
	go conversation.RunUnsnoozer(ctx, unsnoozeInterval)
	go notifier.Run(ctx)
	go sla.Run(ctx, slaEvaluationInterval)
	go sla.SendNotifications(ctx)
	go media.DeleteUnlinkedMedia(ctx)
	go user.MonitorAgentAvailability(ctx)

	var app = &App{
		lo:            lo,
		fs:            fs,
		sla:           sla,
		oidc:          oidc,
		i18n:          i18n,
		auth:          auth,
		media:         media,
		setting:       settings,
		inbox:         inbox,
		user:          user,
		team:          team,
		status:        status,
		priority:      priority,
		tmpl:          template,
		notifier:      notifier,
		consts:        atomic.Value{},
		conversation:  conversation,
		automation:    automation,
		businessHours: businessHours,
		authz:         initAuthz(i18n),
		view:          initView(db),
		csat:          initCSAT(db, i18n),
		search:        initSearch(db, i18n),
		role:          initRole(db, i18n),
		tag:           initTag(db, i18n),
		macro:         initMacro(db, i18n),
		ai:            initAI(db, i18n),
	}
	app.consts.Store(constants)

	g := fastglue.NewGlue()
	g.SetContext(app)
	initHandlers(g, wsHub)

	s := &fasthttp.Server{
		Name:                 appName,
		ReadTimeout:          ko.MustDuration("app.server.read_timeout"),
		WriteTimeout:         ko.MustDuration("app.server.write_timeout"),
		MaxRequestBodySize:   ko.MustInt("app.server.max_body_size"),
		MaxKeepaliveDuration: ko.MustDuration("app.server.keepalive_timeout"),
		ReadBufferSize:       ko.Int("app.server.read_buffer_size"),
	}

	go func() {
		colorlog.Green("Server started at %s", ko.String("app.server.address"))
		if ko.String("server.socket") != "" {
			colorlog.Green("Unix socket created at %s", ko.String("server.socket"))
		}
		if err := g.ListenAndServe(ko.String("app.server.address"), ko.String("server.socket"), s); err != nil {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	// Start the app update checker.
	if ko.Bool("app.check_updates") {
		go checkUpdates(versionString, time.Hour*1, app)
	}

	// Wait for shutdown signal.
	<-ctx.Done()
	colorlog.Red("Shutting down HTTP server...")
	s.Shutdown()
	colorlog.Red("Shutting down inboxes...")
	inbox.Close()
	colorlog.Red("Shutting down automation...")
	automation.Close()
	colorlog.Red("Shutting down autoassigner...")
	autoassigner.Close()
	colorlog.Red("Shutting down notifier...")
	notifier.Close()
	colorlog.Red("Shutting down conversation...")
	conversation.Close()
	colorlog.Red("Shutting down SLA...")
	sla.Close()
	colorlog.Red("Shutting down database...")
	db.Close()
	colorlog.Red("Shutting down redis...")
	rdb.Close()
	colorlog.Green("Shutdown complete.")
}
