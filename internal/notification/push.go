package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	nmodels "github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/ssrf"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const (
	pushPublicKeySetting  = "notification.push.vapid_public_key"
	pushPrivateKeySetting = "notification.push.vapid_private_key"
	pushTTL               = 86400
	pushTitleRunes        = 100
	pushBodyRunes         = 400
)

type pushSender func(context.Context, []byte, PushSubscription, string, string, string) (*http.Response, error)

type pushDelivery struct {
	UserID  int
	Payload nmodels.PushPayload
}

type pushSettingStore interface {
	Get(key string) (types.JSONText, error)
	Update(settings any) error
}

type PushManager struct {
	store       pushSubscriptionStore
	lo          *logf.Logger
	i18n        *i18n.I18n
	publicKey   string
	privateKey  string
	subject     string
	queue       chan pushDelivery
	concurrency int
	sender      pushSender
}

type PushManagerOpts struct {
	DB          *sqlx.DB
	Settings    pushSettingStore
	Lo          *logf.Logger
	I18n        *i18n.I18n
	RootURL     string
	Concurrency int
	QueueSize   int
}

func NewPushManager(opts PushManagerOpts) (*PushManager, error) {
	var q pushQueries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, queriesFS); err != nil {
		return nil, err
	}
	privateKey, publicKey, err := loadVAPIDKeys(opts.Settings)
	if err != nil {
		opts.Lo.Error("error loading VAPID keys, push notifications are off", "error", err)
	}
	m := &PushManager{
		store:       &sqlPushStore{q: q},
		lo:          opts.Lo,
		i18n:        opts.I18n,
		publicKey:   publicKey,
		privateKey:  privateKey,
		subject:     vapidSubject(opts.RootURL),
		queue:       make(chan pushDelivery, opts.QueueSize),
		concurrency: opts.Concurrency,
	}
	pushHTTPClient := newPushHTTPClient(opts.Lo)
	m.sender = func(ctx context.Context, payload []byte, subscription PushSubscription, subject, publicKey, privateKey string) (*http.Response, error) {
		return sendWebPush(ctx, payload, subscription, subject, publicKey, privateKey, pushHTTPClient)
	}
	return m, nil
}

func (m *PushManager) PublicKey() string {
	return m.publicKey
}

func (m *PushManager) Endpoints(userID int) ([]string, error) {
	subscriptions, err := m.store.List(userID)
	if err != nil {
		m.lo.Error("error fetching push subscriptions", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	endpoints := make([]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		endpoints = append(endpoints, subscription.Endpoint)
	}
	return endpoints, nil
}

func (m *PushManager) Upsert(userID int, subscription PushSubscriptionInput) error {
	if !validPushSubscription(subscription) {
		return envelope.NewError(envelope.InputError, m.i18n.T("notification.invalidPushSubscription"), nil)
	}
	if err := m.store.Upsert(userID, subscription); err != nil {
		m.lo.Error("error saving push subscription", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (m *PushManager) Delete(userID int, endpoint string) error {
	if endpoint == "" {
		return envelope.NewError(envelope.InputError, m.i18n.T("notification.invalidPushSubscription"), nil)
	}
	if err := m.store.Delete(userID, endpoint); err != nil {
		m.lo.Error("error deleting push subscription", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (m *PushManager) Send(userID int, payload nmodels.PushPayload) bool {
	if m.publicKey == "" || m.privateKey == "" {
		return false
	}
	select {
	case m.queue <- pushDelivery{UserID: userID, Payload: payload}:
		return true
	default:
		m.lo.Error("push notification queue is full", "user_id", userID)
		return false
	}
}

func (m *PushManager) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range m.concurrency {
		wg.Go(func() { m.worker(ctx) })
	}
	wg.Wait()
}

func (m *PushManager) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery := <-m.queue:
			if err := m.deliver(ctx, delivery); err != nil {
				m.lo.Error("error sending push notification", "user_id", delivery.UserID, "error", err)
			}
		}
	}
}

func (m *PushManager) deliver(ctx context.Context, delivery pushDelivery) error {
	subscriptions, err := m.store.List(delivery.UserID)
	if err != nil {
		return err
	}
	delivery.Payload.Title = pushPreview(delivery.Payload.Title, pushTitleRunes)
	delivery.Payload.Body = pushPreview(delivery.Payload.Body, pushBodyRunes)
	payload, err := json.Marshal(delivery.Payload)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		resp, err := m.sender(ctx, payload, subscription, m.subject, m.publicKey, m.privateKey)
		if err != nil {
			m.lo.Error("push endpoint rejected notification", "subscription_id", subscription.ID, "error", err)
			continue
		}
		if resp == nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			if err := m.store.DeleteByID(subscription.ID); err != nil {
				m.lo.Error("error deleting expired push subscription", "subscription_id", subscription.ID, "error", err)
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			m.lo.Error("push endpoint returned error", "subscription_id", subscription.ID, "status", resp.StatusCode)
		}
	}
	return nil
}

func loadVAPIDKeys(settings pushSettingStore) (string, string, error) {
	privateKey, err := readStringSetting(settings, pushPrivateKeySetting)
	if err != nil {
		return "", "", err
	}
	publicKey, err := readStringSetting(settings, pushPublicKeySetting)
	if err != nil {
		return "", "", err
	}
	if privateKey != "" && publicKey != "" {
		return privateKey, publicKey, nil
	}
	privateKey, publicKey, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", err
	}
	if err := settings.Update(map[string]string{
		pushPrivateKeySetting: privateKey,
		pushPublicKeySetting:  publicKey,
	}); err != nil {
		return "", "", err
	}
	return privateKey, publicKey, nil
}

func readStringSetting(settings pushSettingStore, key string) (string, error) {
	b, err := settings.Get(key)
	if err != nil {
		return "", err
	}
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return "", err
	}
	return value, nil
}

func sendWebPush(ctx context.Context, payload []byte, subscription PushSubscription, subject, publicKey, privateKey string, client *http.Client) (*http.Response, error) {
	return webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256DH,
			Auth:   subscription.Auth,
		},
	}, &webpush.Options{
		HTTPClient:      client,
		Subscriber:      subject,
		VAPIDPublicKey:  publicKey,
		VAPIDPrivateKey: privateKey,
		TTL:             pushTTL,
	})
}

func validPushSubscription(subscription PushSubscriptionInput) bool {
	endpoint, err := url.Parse(subscription.Endpoint)
	return err == nil && endpoint.Scheme == "https" && endpoint.Host != "" && subscription.P256DH != "" && subscription.Auth != ""
}

func vapidSubject(rootURL string) string {
	rootURL = strings.TrimRight(rootURL, "/")
	if strings.HasPrefix(rootURL, "https://") {
		return rootURL
	}
	return "support@libredesk.io"
}

func newPushHTTPClient(lo *logf.Logger) *http.Client {
	control := ssrf.NewControl(true, nil, lo)
	return &http.Client{
		Transport:     ssrf.NewTransport(control, 10*time.Second),
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func pushPreview(text string, limit int) string {
	count := 0
	for i := range text {
		if count == limit {
			return text[:i] + "…"
		}
		count++
	}
	return text
}
