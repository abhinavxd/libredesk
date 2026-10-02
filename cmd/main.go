package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/knadh/go-i18n"
	"hash/fnv"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/jakedolan443/fernmail/internal/address"
	auth_ "github.com/jakedolan443/fernmail/internal/auth"
	"github.com/jakedolan443/fernmail/internal/authz"

	"github.com/jakedolan443/fernmail/internal/colorlog"

	accountmail "github.com/jakedolan443/fernmail/internal/accountmail"

	"github.com/jakedolan443/fernmail/internal/resourceimage"

	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/redis/go-redis/v9"

	"github.com/jakedolan443/fernmail/internal/conversation"

	"github.com/jakedolan443/fernmail/internal/importer"
	"github.com/jakedolan443/fernmail/internal/inbox"
	"github.com/jakedolan443/fernmail/internal/media"
	"github.com/jakedolan443/fernmail/internal/oidc"
	"github.com/jakedolan443/fernmail/internal/ratelimit"
	"github.com/jakedolan443/fernmail/internal/role"
	"github.com/jakedolan443/fernmail/internal/setting"
	"github.com/jakedolan443/fernmail/internal/team"
	"github.com/jakedolan443/fernmail/internal/template"
	"github.com/jakedolan443/fernmail/internal/user"
	"github.com/jakedolan443/fernmail/internal/webhook"
	"github.com/jakedolan443/fernmail/internal/ws"
	wsmodels "github.com/jakedolan443/fernmail/internal/ws/models"

	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko          = koanf.New(".")
	ctx         = context.Background()
	appName     = "fernmail"
	frontendDir = "frontend/dist/main"

	// Injected at build time.
	buildString   string
	versionString string

	// assetVersion is hashed from buildString so public cache-bust URLs don't leak the version.
	assetVersion = func() string {
		src := buildString
		if src == "" {
			src = strconv.FormatInt(time.Now().Unix(), 36)
		}
		h := fnv.New32a()
		h.Write([]byte(src))
		return strconv.FormatUint(uint64(h.Sum32()), 36)
	}()
)

const (
	sampleEncKey = "your-32-char-random-string-here!"

	serverShutdownTimeout = 8 * time.Second
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	resourceImages *resourceimage.Store
	ctx            context.Context
	fs             stuffbin.FileSystem
	consts         atomic.Value
	auth           *auth_.Auth
	authz          *authz.Enforcer
	i18n           *i18n.I18n
	lo             *logf.Logger
	oidc           *oidc.Manager
	media          *media.Manager
	setting        *setting.Manager
	role           *role.Manager
	user           *user.Manager
	team           *team.Manager
	inbox          *inbox.Manager
	address        *address.Manager
	tmpl           *template.Manager
	conversation   *conversation.Manager
	accountmail    *accountmail.Service
	webhook        *webhook.Manager
	rateLimit      *ratelimit.Limiter
	redis          *redis.Client
	fc             *fastcache.FastCache
	importer       *importer.Importer
	wsHub          *ws.Hub

	// Flag to indicate if app restart is required for settings to take effect.
	restartRequired bool
	sync.Mutex
}

func main() {
	// Set up signal handler.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load command line flags into Koanf.
	initFlags()

	if ko.Bool("version") {
		fmt.Println(buildString)
		os.Exit(0)
	}

	// Build string injected at build time.
	colorlog.Green("Build: %s", buildString)

	// Load the config files into Koanf.
	initConfig(ko)

	// Validate custom static directory if provided.
	if dir := getCustomStaticDir(); dir != "" {
		if _, err := os.Stat(dir); err != nil {
			log.Fatalf("--static-dir path not accessible: %s: %v", dir, err)
		}
		colorlog.Green("Using custom static directory: %s", dir)
	}

	// Init stuffbin fs with optional custom static dir overlay.
	fs := initFS(getCustomStaticDir())

	db := initDB()

	if ko.Bool("install") {
		install(ctx, db, fs, ko.Bool("idempotent-install"), !ko.Bool("yes"))
		os.Exit(0)
	}

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

	if ko.Bool("upgrade") {
		upgrade(db, fs, !ko.Bool("yes"))
		os.Exit(0)
	}

	checkPendingUpgrade(db)

	// Load app settings from DB into the Koanf instance.
	settings := initSettings(db)
	loadSettings(settings)
	loadResourceLimits(settings)

	validateConfig(ko)

	startPprof()

	// Fallback for config typo. Logs a warning but continues to work with the incorrect key.
	// Uses 'message.message_outgoing_scan_interval' (correct key) as default key, falls back to the common typo.
	msgOutgoingScanIntervalKey := "message.message_outgoing_scan_interval"
	if ko.String(msgOutgoingScanIntervalKey) == "" {
		if ko.String("message.message_outoing_scan_interval") != "" {
			colorlog.Red("WARNING: typo in config key 'message.message_outoing_scan_interval' detected. Use 'message.message_outgoing_scan_interval' instead in your config.toml file. Support for this incorrect key will be removed in a future release.")
			msgOutgoingScanIntervalKey = "message.message_outoing_scan_interval"
		}
	}

	var (
		draftRetentionDuration      = cmp.Or(ko.Duration("conversation.draft_retention_duration"), 360*time.Hour)
		messageOutgoingQWorkers     = ko.MustDuration("message.outgoing_queue_workers")
		messageIncomingQWorkers     = ko.MustDuration("message.incoming_queue_workers")
		messageOutgoingScanInterval = ko.MustDuration(msgOutgoingScanIntervalKey)
		lo                          = initLogger(appName)
		rdb                         = initRedis()
		constants                   = initConstants()
		i18n                        = initI18n(fs)
		oidc                        = initOIDC(db, settings, i18n)
		ssrfControl                 = initSSRFControl()
		auth                        = initAuth(oidc, rdb, i18n, ssrfControl)
		template                    = initTemplate(db, fs, constants, i18n)
		media                       = initMedia(db, i18n, settings)
		resourceImages              = resourceimage.NewStore(db, media, constants.UploadProvider, settings.GetResourcePolicyTx)
		inbox                       = initInbox(db, i18n)
		address                     = initAddress(db)
		team                        = initTeam(db, i18n)
		webhook                     = initWebhook(db, i18n, ssrfControl)
		user                        = initUser(i18n, db)
		wsHub                       = initWS(user)
		accountmail                 = initAccountMailer()
		conversation                = initConversations(i18n, wsHub, db, inbox, user, media, settings, template, webhook, resourceImages)
		rateLimiter                 = initRateLimit(rdb)
	)

	media.SetStorageFullNotifier(func() {
		data, err := json.Marshal(wsmodels.Message{
			Type: wsmodels.MessageTypeSystemToast,
			Data: map[string]string{
				"variant":     "destructive",
				"message_key": "media.storageFull",
				"description": "Durable media storage is full. New uploads are temporarily disabled.",
			},
		})
		if err != nil {
			lo.Error("error marshalling storage-full toast", "error", err)
			return
		}
		wsHub.BroadcastMessage(wsmodels.BroadcastMessage{Data: data})
	})

	wsHub.SetConversationStore(conversation)

	startInboxes(ctx, inbox, conversation, user, conversation.SignAvatarURL)

	go conversation.Run(ctx, messageIncomingQWorkers, messageOutgoingQWorkers, messageOutgoingScanInterval)
	go webhook.Run(ctx)
	go accountmail.Run(ctx)
	go media.DeleteUnlinkedMedia(ctx)
	go resourceImages.RunCacheCleaner(ctx, func(err error) { lo.Error("error cleaning external image cache", "error", err) })
	go user.MonitorUserAvailability(ctx, onUsersOffline(conversation))
	go conversation.RunDraftCleaner(ctx, draftRetentionDuration)

	var app = &App{
		ctx:            ctx,
		lo:             lo,
		fs:             fs,
		oidc:           oidc,
		i18n:           i18n,
		auth:           auth,
		media:          media,
		setting:        settings,
		inbox:          inbox,
		address:        address,
		user:           user,
		team:           team,
		tmpl:           template,
		accountmail:    accountmail,
		consts:         atomic.Value{},
		conversation:   conversation,
		authz:          initAuthz(i18n, inbox, address),
		role:           initRole(db, i18n),
		importer:       initImporter(i18n),
		webhook:        webhook,
		rateLimit:      rateLimiter,
		redis:          rdb,
		resourceImages: resourceImages,
		fc:             initFastCache(rdb),
		wsHub:          wsHub,
	}
	app.consts.Store(constants)

	g := fastglue.NewGlue()
	g.SetContext(app)
	initHandlers(g, wsHub)

	// Buffers above this are dropped rather than reused, and the ones we keep stay with the connection until it closes.
	fasthttp.SetBodySizePoolLimit(64<<10, 128<<10) // request: 64 KiB, response: 128 KiB

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
		if ko.String("app.server.socket") != "" {
			colorlog.Green("Unix socket created at %s", ko.String("app.server.socket"))
		}
		if err := g.ListenAndServe(ko.String("app.server.address"), ko.String("app.server.socket"), s); err != nil {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	// Wait for shutdown signal.
	<-ctx.Done()
	closedAgentConns := wsHub.CloseAll()
	colorlog.Red("Closed %d mailbox websocket connections.", closedAgentConns)
	colorlog.Red("Shutting down HTTP server...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
	if err := s.ShutdownWithContext(shutdownCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			colorlog.Red("HTTP server drain timed out after %s, exiting with connections still open: %v", serverShutdownTimeout, err)
		} else {
			colorlog.Red("error shutting down HTTP server: %v", err)
		}
	}
	cancelShutdown()
	colorlog.Red("Shutting down inboxes...")
	inbox.Close()
	colorlog.Red("Shutting down accountmail...")
	accountmail.Close()
	colorlog.Red("Shutting down webhook...")
	webhook.Close()
	colorlog.Red("Shutting down conversation...")
	conversation.Close()
	colorlog.Red("Shutting down importer...")
	app.importer.Close()
	colorlog.Red("Shutting down database...")
	db.Close()
	colorlog.Red("Shutting down redis...")
	rdb.Close()
	colorlog.Green("Shutdown complete.")
}

// onUsersOffline returns a callback for MonitorUserAvailability that broadcasts
// offline status to the appropriate clients based on user type.
func onUsersOffline(conv *conversation.Manager) func([]umodels.OfflineUser) {
	return func(users []umodels.OfflineUser) {
		for _, u := range users {
			switch u.Type {
			case umodels.UserTypeAgent:
				conv.BroadcastAgentAvailability(u.ID, umodels.Offline)
			}
		}
	}
}
