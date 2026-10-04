package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"github.com/knadh/go-i18n"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"html/template"

	"github.com/jakedolan443/fernmail/internal/activationkey"
	"github.com/jakedolan443/fernmail/internal/address"
	auth_ "github.com/jakedolan443/fernmail/internal/auth"
	"github.com/jakedolan443/fernmail/internal/authz"

	"github.com/jakedolan443/fernmail/internal/colorlog"

	"github.com/jakedolan443/fernmail/internal/conversation"

	"github.com/jakedolan443/fernmail/internal/importer"
	"github.com/jakedolan443/fernmail/internal/inbox"
	"github.com/jakedolan443/fernmail/internal/inbox/channel/email"

	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"

	accountmail "github.com/jakedolan443/fernmail/internal/accountmail"
	emailaccountmail "github.com/jakedolan443/fernmail/internal/accountmail/providers/email"
	"github.com/jakedolan443/fernmail/internal/media"
	fs "github.com/jakedolan443/fernmail/internal/media/stores/localfs"
	"github.com/jakedolan443/fernmail/internal/media/stores/s3"

	"github.com/jakedolan443/fernmail/internal/oidc"
	"github.com/jakedolan443/fernmail/internal/ratelimit"

	"github.com/jakedolan443/fernmail/internal/resourceimage"
	"github.com/jakedolan443/fernmail/internal/role"
	"github.com/jakedolan443/fernmail/internal/setting"

	"github.com/jakedolan443/fernmail/internal/ssrf"
	"github.com/jakedolan443/fernmail/internal/team"
	tmpl "github.com/jakedolan443/fernmail/internal/template"
	"github.com/jakedolan443/fernmail/internal/user"
	"github.com/jakedolan443/fernmail/internal/webhook"
	"github.com/jakedolan443/fernmail/internal/ws"
	"github.com/jmoiron/sqlx"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	flag "github.com/spf13/pflag"
	"github.com/volatiletech/null/v9"
	goredis "github.com/zerodha/fastcache/stores/goredis/v9"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/logf"
)

// Branding used when the site title or logo is unset.
const (
	defaultSiteName   = "Fernmail"
	defaultFaviconURL = "/favicon.svg"
)

// constants holds the app constants.
type constants struct {
	AppBaseURL string
	// FaviconURL is the site logo, or the bundled favicon when no logo is set.
	FaviconURL string
	// LogoURL is absolute so it also resolves in emails; empty when unset.
	LogoURL                     string
	SiteName                    string
	UploadProvider              string
	AllowedUploadFileExtensions []string
	MaxFileUploadSizeMB         int
}

// Config loads config from files and environment variables into koanf.
func initConfig(ko *koanf.Koanf) {
	for _, f := range ko.Strings("config") {
		log.Println("reading config file:", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				log.Printf("WARNING: Config file not found. Continuing with defaults and environment variables.")
				continue
			}
			log.Fatalf("error loading config from file: %v.", err)
		}
	}
	// Load environment variables with `LIBREDESK_` prefix.
	ko.Load(env.Provider(".", env.Opt{
		Prefix: "LIBREDESK_",
		TransformFunc: func(key, val string) (string, any) {
			// Transform the key.
			key = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(key, "LIBREDESK_")), "__", ".")
			return key, val
		},
	}), nil)
}

// validateConfig logs warnings/fatals for invalid config values.
func validateConfig(ko *koanf.Koanf) {
	encKey := ko.MustString("app.encryption_key")

	if len(encKey) != 32 {
		log.Fatalf("encryption_key must be exactly 32 characters, got %d", len(encKey))
	}

	// Warn if using sample config value.
	if encKey == sampleEncKey {
		colorlog.Red("WARNING: You are using the sample encryption_key from config.sample.toml. Change it immediately. Generate a secure key with `openssl rand -hex 16`")
	}
}

// initFlags initializes the commandline flags.
func initFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)

	// Registering `--help` handler.
	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		f.PrintDefaults()
	}

	// Register the commandline flags and parse them.
	f.StringSlice("config", []string{"config.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("version", false, "show current version of the build")
	f.Bool("install", false, "setup database")
	f.Bool("idempotent-install", false, "run idempotent installation, i.e., skip installion if schema is already installed useful for the first time setup")
	f.Bool("yes", false, "skip confirmation prompt")
	f.Bool("upgrade", false, "upgrade the database schema")
	f.Bool("set-system-user-password", false, "set password for the system user")
	f.String("static-dir", "", "path to a directory with custom static files and templates to override the defaults")

	if err := f.Parse(os.Args[1:]); err != nil {
		log.Fatalf("loading flags: %v", err)
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		log.Fatalf("loading config: %v", err)
	}
}

// initConstants initializes the app constants.
func initConstants() *constants {
	rootURL := ko.String("app.root_url")
	logoURL := absoluteAppURL(rootURL, strings.TrimSpace(ko.String("app.logo_url")))
	return &constants{
		AppBaseURL:                  rootURL,
		FaviconURL:                  cmp.Or(logoURL, defaultFaviconURL),
		LogoURL:                     logoURL,
		SiteName:                    cmp.Or(strings.TrimSpace(ko.String("app.site_name")), defaultSiteName),
		UploadProvider:              ko.MustString("upload.provider"),
		AllowedUploadFileExtensions: ko.Strings("app.allowed_file_upload_extensions"),
		MaxFileUploadSizeMB:         ko.Int("app.max_file_upload_size"),
	}
}

// absoluteAppURL prefixes an app-relative path, such as an uploaded logo, with the root URL.
func absoluteAppURL(rootURL, u string) string {
	if strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//") {
		return strings.TrimRight(rootURL, "/") + u
	}
	return u
}

// initFS initializes the stuffbin FileSystem. If staticDir is set, files from
// that directory are merged into the FS, overriding embedded defaults.
func initFS(staticDir string) stuffbin.FileSystem {
	// Custom static dir mirrors the structure of the built-in static/ directory.
	staticFiles := []string{
		"./:static",
	}

	// Get self executable path.
	path, err := os.Executable()
	if err != nil {
		log.Fatalf("error initializing FS: %v", err)
	}

	// Load embedded files in the executable.
	fs, err := stuffbin.UnStuff(path)

	if err != nil {
		if err == stuffbin.ErrNoID {
			// Running in local/dev mode, use the local filesystem.
			// Only include frontend dirs if they exist (frontend build is optional in dev).
			colorlog.Red("binary unstuff failed, using local filesystem for static files")
			files := []string{"i18n", "static", "schema.sql"}
			for _, d := range []string{"frontend/dist/main"} {
				if _, err := os.Stat(d); err == nil {
					files = append(files, d)
				}
			}
			fs, err = stuffbin.NewLocalFS("/", files...)
			if err != nil {
				log.Fatalf("error initializing local FS: %v", err)
			}
		} else {
			log.Fatalf("error initializing FS: %v", err)
		}
	}

	// Merge custom static files if a custom static dir is provided.
	if staticDir != "" {
		// Only include paths that exist in the custom dir.
		var sf []string
		for _, def := range staticFiles {
			src, _, _ := strings.Cut(def, ":")
			if _, err := os.Stat(filepath.Join(staticDir, src)); err == nil {
				sf = append(sf, def)
			}
		}

		if len(sf) > 0 {
			files := joinFSPaths(staticDir, sf)
			fLocal, err := stuffbin.NewLocalFS("/", files...)
			if err != nil {
				log.Fatalf("error loading custom static files from '%s': %v", staticDir, err)
			}
			if err := fs.Merge(fLocal); err != nil {
				log.Fatalf("error merging custom static files from '%s': %v", staticDir, err)
			}
		}
	}

	return fs
}

// joinFSPaths joins a root directory with stuffbin path specs (local:virtual).
func joinFSPaths(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		f := strings.Split(p, ":")
		out = append(out, filepath.Join(root, f[0])+":"+f[1])
	}
	return out
}

// loadSettings loads settings from the DB into Koanf map.
func loadSettings(m *setting.Manager) {
	j, err := m.GetAllJSON()
	if err != nil {
		log.Fatalf("error parsing settings from DB: %v", err)
	}

	// Setting keys are dot separated, eg: app.logo_url. Unflatten them into
	// nested maps {app: {logo_url}}.
	var out map[string]any

	if err := json.Unmarshal(j, &out); err != nil {
		log.Fatalf("error unmarshalling settings from DB: %v", err)
	}
	if err := ko.Load(confmap.Provider(out, "."), nil); err != nil {
		log.Fatalf("error parsing settings from DB: %v", err)
	}
}

// initSettings inits setting manager.
func initSettings(db *sqlx.DB) *setting.Manager {
	s, err := setting.New(setting.Opts{
		DB:            db,
		Lo:            initLogger("settings"),
		EncryptionKey: ko.MustString("app.encryption_key"),
	})
	if err != nil {
		log.Fatalf("error initializing setting manager: %v", err)
	}
	return s
}

// initUser inits user manager.
func initUser(i18n *i18n.I18n, DB *sqlx.DB) *user.Manager {
	mgr, err := user.New(i18n, user.Opts{
		DB: DB,
		Lo: initLogger("user_manager"),
	})
	if err != nil {
		log.Fatalf("error initializing user manager: %v", err)
	}
	return mgr
}

// initConversations inits conversation manager.
func initConversations(
	i18n *i18n.I18n,
	hub *ws.Hub,
	db *sqlx.DB,
	inboxStore *inbox.Manager,
	userStore *user.Manager,
	mediaStore *media.Manager,
	settings *setting.Manager,
	template *tmpl.Manager,
	webhook *webhook.Manager,
	resourceImages *resourceimage.Store,
) *conversation.Manager {

	c, err := conversation.New(hub, i18n, inboxStore, userStore, mediaStore, settings, template, webhook, conversation.Opts{
		CacheIncomingImages: func(ctx context.Context, id int, content string) error {
			return resourceImages.Prefetch(ctx, id, content, settings.GetResourcePolicyTx)
		},
		DB:                       db,
		Lo:                       initLogger("conversation_manager"),
		OutgoingMessageQueueSize: ko.MustInt("message.outgoing_queue_size"),
		IncomingMessageQueueSize: ko.MustInt("message.incoming_queue_size"),
		SubjectRefFormat:         ko.String("conversation.subject_ref_format"),
	})
	if err != nil {
		log.Fatalf("error initializing conversation manager: %v", err)
	}
	return c
}

// initActivationKeys inits the Key Distribution pools.
func initActivationKeys(db *sqlx.DB) *activationkey.Manager {
	m, err := activationkey.New(activationkey.Opts{
		DB:            db,
		Lo:            initLogger("activation_keys"),
		EncryptionKey: ko.MustString("app.encryption_key"),
	})
	if err != nil {
		log.Fatalf("error initializing activation keys: %v", err)
	}
	return m
}

func initAddress(db *sqlx.DB) *address.Manager {
	m, err := address.New(db)
	if err != nil {
		log.Fatalf("error initializing address manager: %v", err)
	}
	return m
}

// initWS inits websocket hub.
func initWS(user *user.Manager) *ws.Hub {
	return ws.NewHub(initLogger("ws"), user)
}

// getCustomStaticDir returns the custom static directory path from CLI flag or config.
func getCustomStaticDir() string {
	dir := ko.String("static-dir")
	if dir == "" {
		dir = ko.String("app.static_dir")
	}
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

// initTemplates inits template manager.
func initTemplate(db *sqlx.DB, fs stuffbin.FileSystem, consts *constants, i18n *i18n.I18n) *tmpl.Manager {
	var (
		lo      = initLogger("template")
		funcMap = getTmplFuncs(consts, i18n, fs)
	)
	tpls, err := stuffbin.ParseTemplatesGlob(funcMap, fs, "/static/email-templates/*.html")
	if err != nil {
		log.Fatalf("error parsing e-mail templates: %v", err)
	}
	webTpls, err := parseWebTemplates(funcMap, fs)
	if err != nil {
		log.Fatalf("error parsing web templates: %v", err)
	}

	m, err := tmpl.New(lo, db, webTpls, tpls, funcMap, i18n)
	if err != nil {
		log.Fatalf("error initializing template manager: %v", err)
	}
	return m
}

// getTmplFuncs returns the template functions.
func getTmplFuncs(consts *constants, i18n *i18n.I18n, fs stuffbin.FileSystem) template.FuncMap {
	return template.FuncMap{
		"RootURL": func() string {
			return consts.AppBaseURL
		},
		"FaviconURL": func() string {
			return consts.FaviconURL
		},
		"AssetVer": func() string {
			return assetVersion
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
		"LogoURL": func() string {
			return consts.LogoURL
		},
		"SiteName": func() string {
			return consts.SiteName
		},
		"L": func() any {
			return i18n
		},
		"map": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("map: odd number of arguments")
			}
			out := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				key, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("map: key %v is not a string", pairs[i])
				}
				out[key] = pairs[i+1]
			}
			return out, nil
		},
	}
}

// reloadSettings reloads the settings from the database into the Koanf instance.
func reloadSettings(app *App) error {
	app.lo.Info("reloading settings")
	j, err := app.setting.GetAllJSON()
	if err != nil {
		app.lo.Error("error parsing settings from DB", "error", err)
		return err
	}
	var out map[string]any
	if err := json.Unmarshal(j, &out); err != nil {
		app.lo.Error("error unmarshalling settings from DB", "error", err)
		return err
	}
	app.Lock()
	err = ko.Load(confmap.Provider(out, "."), nil)
	app.Unlock()
	if err != nil {
		app.lo.Error("error loading settings into koanf", "error", err)
		return err
	}
	newConsts := initConstants()
	app.consts.Store(newConsts)
	return nil
}

// reloadTemplates reloads the templates from the filesystem.
func reloadTemplates(app *App) error {
	app.lo.Info("reloading templates")
	funcMap := getTmplFuncs(app.consts.Load().(*constants), app.i18n, app.fs)
	tpls, err := stuffbin.ParseTemplatesGlob(funcMap, app.fs, "/static/email-templates/*.html")
	if err != nil {
		app.lo.Error("error parsing email templates", "error", err)
		return err
	}
	webTpls, err := parseWebTemplates(funcMap, app.fs)
	if err != nil {
		app.lo.Error("error parsing web templates", "error", err)
		return err
	}

	return app.tmpl.Reload(webTpls, tpls, funcMap)
}

// parseWebTemplates parses the top-level web templates.
func parseWebTemplates(funcMap template.FuncMap, fs stuffbin.FileSystem) (*template.Template, error) {
	var paths []string
	for _, pattern := range []string{
		"/static/public/web-templates/*.html",
	} {
		p, err := fs.Glob(pattern)
		if err != nil {
			return nil, err
		}
		paths = append(paths, p...)
	}
	return stuffbin.ParseTemplates(funcMap, fs, paths...)
}

// initTeam inits team manager.
func initTeam(db *sqlx.DB, i18n *i18n.I18n) *team.Manager {
	var lo = initLogger("team-manager")
	mgr, err := team.New(team.Opts{
		DB:   db,
		Lo:   lo,
		I18n: i18n,
	})
	if err != nil {
		log.Fatalf("error initializing team manager: %v", err)
	}
	return mgr
}

// initMedia inits media manager.
func initMedia(db *sqlx.DB, i18n *i18n.I18n, settings *setting.Manager) *media.Manager {
	var (
		store media.Store
		err   error
		lo    = initLogger("media")
	)
	maxStorageBytes := liveResourceLimits.snapshot().MaxStorageBytes
	if maxStorageBytes < 0 {
		log.Fatalf("upload.max_storage_bytes must be zero or a positive number of bytes")
	}
	rootURL := func() string {
		u, err := settings.GetAppRootURL()
		if err != nil {
			// Fallback to config if settings fetch fails
			return ko.String("app.root_url")
		}
		return u
	}
	switch s := ko.MustString("upload.provider"); s {
	case "s3":
		store, err = s3.New(s3.Opt{
			URL:        ko.String("upload.s3.url"),
			PublicURL:  ko.String("upload.s3.public_url"),
			AccessKey:  ko.String("upload.s3.access_key"),
			SecretKey:  ko.String("upload.s3.secret_key"),
			Region:     ko.String("upload.s3.region"),
			Bucket:     ko.String("upload.s3.bucket"),
			BucketPath: ko.String("upload.s3.bucket_path"),
			// All files are private by default.
			BucketType: "private",
			Expiry:     ko.Duration("upload.s3.expiry"),
		})
		if err != nil {
			log.Fatalf("error initializing s3 media store: %v", err)
		}
	case "fs":
		// Default expiry to 1h if not set.
		fsExpiry := ko.Duration("upload.fs.expiry")
		if fsExpiry == 0 {
			fsExpiry = 1 * time.Hour
		}
		store, err = fs.New(fs.Opts{
			UploadURI:  media.PublicURI,
			UploadPath: filepath.Clean(ko.String("upload.fs.upload_path")),
			RootURL:    rootURL,
			SigningKey: ko.MustString("app.encryption_key"),
			Expiry:     fsExpiry,
		})
		if err != nil {
			log.Fatalf("error initializing fs media store: %v", err)
		}
	default:
		log.Fatalf("unknown media store: %s", s)
	}

	media, err := media.New(media.Opts{
		Store:           store,
		Lo:              lo,
		DB:              db,
		I18n:            i18n,
		RootURL:         rootURL,
		SigningKey:      ko.MustString("app.encryption_key"),
		URLExpiry:       cmp.Or(ko.Duration("upload.fs.expiry"), time.Hour),
		MaxStorageBytes: maxStorageBytes,
	})
	if err != nil {
		log.Fatalf("error initializing media: %v", err)
	}
	return media
}

// initInbox initializes the inbox manager without registering inboxes.
func initInbox(db *sqlx.DB, i18n *i18n.I18n) *inbox.Manager {
	var lo = initLogger("inbox-manager")
	mgr, err := inbox.New(lo, db, i18n, ko.MustString("app.encryption_key"))
	if err != nil {
		log.Fatalf("error initializing inbox manager: %v", err)
	}
	return mgr
}

// initAccountMailer initializes the accountmail service with available providers.
func initAccountMailer() *accountmail.Service {
	smtpCfg := imodels.SMTPConfig{}
	if err := ko.UnmarshalWithConf("account_email", &smtpCfg, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		log.Fatalf("error unmarshalling account email provider config: %v", err)
	}

	emailNotifier, err := emailaccountmail.New([]imodels.SMTPConfig{smtpCfg}, emailaccountmail.Opts{
		Lo:        initLogger("email-accountmail"),
		FromEmail: ko.String("account_email.email_address"),
	})
	if err != nil {
		log.Fatalf("error initializing email accountmail: %v", err)
	}

	accountmailProviders := map[string]accountmail.Notifier{
		emailNotifier.Name(): emailNotifier,
	}

	workers, capacity := ko.Int("account_email.concurrency"), ko.Int("account_email.queue_size")
	if workers < 1 {
		workers = 2
	}
	if capacity < 1 {
		capacity = 2000
	}
	return accountmail.NewService(accountmailProviders, workers, capacity, initLogger("accountmail"))
}

// initEmailInbox loads inbox config from DB and initializes the email inbox.
func initEmailInbox(inboxRecord imodels.Inbox, msgStore inbox.MessageStore, usrStore inbox.UserStore, mgr *inbox.Manager) (inbox.Inbox, error) {
	var config imodels.Config

	// Inbox configuration must never mutate or inherit another inbox's global
	// application configuration (OAuth, aliases and reply routing are private).
	if err := json.Unmarshal(inboxRecord.Config, &config); err != nil {
		return nil, fmt.Errorf("decoding email inbox %s: %w", inboxRecord.Name, err)
	}

	if len(config.SMTP) == 0 {
		log.Printf("WARNING: Zero SMTP servers configured for `%s` inbox: Name: `%s`", inboxRecord.Channel, inboxRecord.Name)
	}

	if len(config.IMAP) == 0 {
		log.Printf("WARNING: Zero IMAP clients configured for `%s` inbox: Name: `%s`", inboxRecord.Channel, inboxRecord.Name)
	}

	config.From = inboxRecord.From
	config.FromNameTemplate = inboxRecord.FromNameTemplate

	if len(config.From) == 0 {
		log.Printf("WARNING: No `from` email address set for `%s` inbox: Name: `%s`", inboxRecord.Channel, inboxRecord.Name)
	}

	maxIncomingMessageSize := liveResourceLimits.snapshot().MaxIncomingMessageSize

	// Callback to persist refreshed tokens in DB.
	tokenRefreshCallback := func(inboxID int, updatedConfig imodels.Config) error {
		// Marshal updated config to JSON
		updatedConfigJSON, err := json.Marshal(updatedConfig)
		if err != nil {
			log.Printf("ERROR: Failed to marshal updated config during token refresh for inbox %d: %v", inboxID, err)
			return err
		}

		// Persist updated config to DB
		if err := mgr.UpdateConfig(inboxID, updatedConfigJSON); err != nil {
			log.Printf("ERROR: Failed to persist refreshed tokens during operation for inbox %d: %v", inboxID, err)
			return err
		}

		log.Printf("INFO: Successfully persisted refreshed tokens during operation for inbox: %d", inboxID)
		return nil
	}

	inbox, err := email.New(msgStore, usrStore, email.Opts{
		ID:                     inboxRecord.ID,
		Name:                   inboxRecord.Name,
		Config:                 config,
		MaxIncomingMessageSize: maxIncomingMessageSize,
		Lo:                     initLogger("email_inbox"),
		TokenRefreshCallback:   tokenRefreshCallback,
	})

	if err != nil {
		return nil, fmt.Errorf("initializing `%s` inbox: `%s` error : %w", inboxRecord.Channel, inboxRecord.Name, err)
	}

	log.Printf("`%s` inbox successfully initialized", inboxRecord.Name)

	return inbox, nil
}

// makeInboxInitializer creates an inbox initializer function.
func makeInboxInitializer(mgr *inbox.Manager, signAvatarURL func(*null.String)) func(imodels.Inbox, inbox.MessageStore, inbox.UserStore) (inbox.Inbox, error) {
	return func(inboxR imodels.Inbox, msgStore inbox.MessageStore, usrStore inbox.UserStore) (inbox.Inbox, error) {
		switch inboxR.Channel {
		case inbox.ChannelEmail:
			return initEmailInbox(inboxR, msgStore, usrStore, mgr)
		default:
			return nil, fmt.Errorf("unknown inbox channel: %s", inboxR.Channel)
		}
	}
}

// reloadInbox reloads a single inbox by ID using the signal-aware context.
func reloadInbox(app *App, id int) error {
	app.lo.Info("reloading inbox", "id", id)
	return app.inbox.ReloadInbox(app.ctx, id, makeInboxInitializer(app.inbox, app.conversation.SignAvatarURL))
}

// startInboxes registers the active inboxes and starts receiver for each.
func startInboxes(ctx context.Context, mgr *inbox.Manager, msgStore inbox.MessageStore, usrStore inbox.UserStore, signAvatarURL func(*null.String)) {
	mgr.SetMessageStore(msgStore)
	mgr.SetUserStore(usrStore)

	if err := mgr.InitInboxes(makeInboxInitializer(mgr, signAvatarURL)); err != nil {
		log.Fatalf("error initializing inboxes: %v", err)
	}

	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("error starting inboxes: %v", err)
	}
}

// initAuthz initializes authorization enforcer.
func initAuthz(i18n *i18n.I18n, inbox *inbox.Manager, address *address.Manager) *authz.Enforcer {
	enforcer, err := authz.NewEnforcer(initLogger("authz"), i18n, inbox.CanAccess, address.CanAccess)
	if err != nil {
		log.Fatalf("error initializing authz: %v", err)
	}
	return enforcer
}

// initSSRFControl builds the shared outbound-request guard from config.
func initSSRFControl() ssrf.Control {
	lo := initLogger("ssrf")
	return ssrf.NewControl(ko.Bool("ssrf.enabled"), ko.Strings("ssrf.allowed_cidrs"), lo)
}

// initAuth initializes the authentication manager.
func initAuth(o *oidc.Manager, rd *redis.Client, i18n *i18n.I18n, dialControl ssrf.Control) *auth_.Auth {
	lo := initLogger("auth")

	providers, err := buildProviders(o)
	if err != nil {
		log.Fatalf("error initializing auth: %v", err)
	}

	secure := !ko.Bool("app.server.disable_secure_cookies")
	sessionLifetime := ko.Duration("app.server.session_lifetime")
	auth, err := auth_.New(auth_.Config{Providers: providers, SecureCookies: secure, SessionLifetime: sessionLifetime}, i18n, rd, lo, dialControl)
	if err != nil {
		log.Fatalf("error initializing auth: %v", err)
	}

	return auth
}

// reloadAuth reloads the auth providers.
func reloadAuth(app *App) error {
	app.lo.Info("reloading auth manager")
	providers, err := buildProviders(app.oidc)
	if err != nil {
		app.lo.Error("error reloading auth", "error", err)
		return err
	}
	if err := app.auth.Reload(auth_.Config{Providers: providers}); err != nil {
		app.lo.Error("error reloading auth", "error", err)
		return err
	}
	return nil
}

// buildProviders creates a list of auth providers from the OIDC manager.
func buildProviders(o *oidc.Manager) ([]auth_.Provider, error) {
	oidcConfigs, err := o.GetAll()
	if err != nil {
		return nil, err
	}

	providers := make([]auth_.Provider, 0, len(oidcConfigs))
	for _, config := range oidcConfigs {
		if !config.Enabled {
			continue
		}
		providers = append(providers, auth_.Provider{
			ID:           config.ID,
			Provider:     config.Provider,
			ProviderURL:  config.ProviderURL,
			RedirectURL:  func() (string, error) { return o.RedirectURL(config.ID) },
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
		})
	}
	return providers, nil
}

// initOIDC initializes open id connect config manager.
func initOIDC(db *sqlx.DB, settings *setting.Manager, i18n *i18n.I18n) *oidc.Manager {
	lo := initLogger("oidc")
	o, err := oidc.New(oidc.Opts{
		DB:            db,
		Lo:            lo,
		I18n:          i18n,
		EncryptionKey: ko.MustString("app.encryption_key"),
	}, settings)
	if err != nil {
		log.Fatalf("error initializing oidc: %v", err)
	}
	return o
}

// initI18n inits i18n.
func initI18n(fs stuffbin.FileSystem) *i18n.I18n {
	lang := cmp.Or(ko.String("app.lang"), defLang)
	log.Printf("loading i18n language file: %s", lang)
	if _, err := fs.Read("/i18n/" + lang + ".json"); err != nil {
		log.Fatalf("error reading i18n language file `%s` : %v", lang, err)
	}
	i18n, err := loadI18nLang(lang, fs)
	if err != nil {
		log.Fatalf("error initializing i18n: %v", err)
	}
	return i18n
}

// initRedis inits redis DB.
func initRedis() *redis.Client {
	// Load options from redis URL if set.
	redisURL := ko.String("redis.url")
	if redisURL != "" {
		options, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("error parsing redis url: %v", err)
		}
		return redis.NewClient(options)
	}
	// Load from individual config options.
	return redis.NewClient(&redis.Options{
		Addr:     ko.MustString("redis.address"),
		Username: ko.String("redis.user"),
		Password: ko.String("redis.password"),
		DB:       ko.Int("redis.db"),
	})
}

func initFastCache(rdb *redis.Client) *fastcache.FastCache {
	return fastcache.New(goredis.New(goredis.Config{Prefix: "mail"}, rdb))
}

// initRedis inits postgres DB.
func initDB() *sqlx.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s %s",
		ko.MustString("db.host"),
		ko.MustInt("db.port"),
		ko.MustString("db.user"),
		ko.MustString("db.password"),
		ko.MustString("db.database"),
		ko.String("db.ssl_mode"),
		ko.String("db.params"),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("error connecting to DB: %v", err)
	}

	db.SetMaxOpenConns(ko.MustInt("db.max_open"))
	db.SetMaxIdleConns(ko.MustInt("db.max_idle"))
	db.SetConnMaxLifetime(ko.MustDuration("db.max_lifetime"))

	return db
}

// initRedis inits role manager.
func initRole(db *sqlx.DB, i18n *i18n.I18n) *role.Manager {
	var lo = initLogger("role_manager")
	r, err := role.New(role.Opts{
		DB:   db,
		Lo:   lo,
		I18n: i18n,
	})
	if err != nil {
		log.Fatalf("error initializing role manager: %v", err)
	}
	return r
}

// initWebhook inits webhook manager.
func initWebhook(db *sqlx.DB, i18n *i18n.I18n, dialControl ssrf.Control) *webhook.Manager {
	var lo = initLogger("webhook")
	m, err := webhook.New(webhook.Opts{
		DB:            db,
		Lo:            lo,
		I18n:          i18n,
		Workers:       ko.MustInt("webhook.workers"),
		QueueSize:     ko.MustInt("webhook.queue_size"),
		Timeout:       ko.MustDuration("webhook.timeout"),
		EncryptionKey: ko.MustString("app.encryption_key"),
		DialControl:   dialControl,
	})
	if err != nil {
		log.Fatalf("error initializing webhook manager: %v", err)
	}
	return m
}

// initImporter inits the importer manager.
func initImporter(i18n *i18n.I18n) *importer.Importer {
	return importer.New(importer.Opts{
		Lo:   initLogger("importer"),
		I18n: i18n,
	})
}

// initLogger initializes a logf logger.
func initLogger(src string) *logf.Logger {
	lvl, env := ko.MustString("app.log_level"), ko.MustString("app.env")
	lo := logf.New(logf.Opts{
		Level:                getLogLevel(lvl),
		EnableColor:          getColor(env),
		EnableCaller:         true,
		CallerSkipFrameCount: 3,
		DefaultFields:        []any{"sc", src},
	})
	return &lo
}

func getColor(env string) bool {
	color := false
	if env == "dev" {
		color = true
	}
	return color
}

func getLogLevel(lvl string) logf.Level {
	switch lvl {
	case "info":
		return logf.InfoLevel
	case "debug":
		return logf.DebugLevel
	case "warn":
		return logf.WarnLevel
	case "error":
		return logf.ErrorLevel
	case "fatal":
		return logf.FatalLevel
	default:
		return logf.InfoLevel
	}
}

// initRateLimit initializes the rate limiter with default rules.
// Defaults are used unless overridden in config.toml under [rate_limit.<name>].
func initRateLimit(redisClient *redis.Client) *ratelimit.Limiter {
	limiter := ratelimit.New(redisClient)

	defaults := []struct {
		Name string
		RPM  int
	}{
		{"auth", 30},
		{"media", 300},
	}

	for _, d := range defaults {
		enabled := true
		rpm := d.RPM

		cfgKey := "rate_limit." + d.Name
		if ko.Exists(cfgKey) {
			var cfg struct {
				Enabled           bool `toml:"enabled"`
				RequestsPerMinute int  `toml:"requests_per_minute"`
			}
			ko.UnmarshalWithConf(cfgKey, &cfg, koanf.UnmarshalConf{Tag: "toml"})
			enabled = cfg.Enabled
			if cfg.RequestsPerMinute > 0 {
				rpm = cfg.RequestsPerMinute
			}
		}

		if !enabled {
			log.Printf("WARNING: rate limit rule '%s' is disabled", d.Name)
		}

		limiter.AddRule(ratelimit.Rule{
			Name:              d.Name,
			Enabled:           enabled,
			RequestsPerMinute: rpm,
		})
	}

	return limiter
}
