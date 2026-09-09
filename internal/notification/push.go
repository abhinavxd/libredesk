package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
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
)

type pushSender func(context.Context, []byte, PushSubscription, string, string, string) (*http.Response, error)

type pushQueries struct {
	List       *sqlx.Stmt `query:"get-push-subscriptions"`
	Upsert     *sqlx.Stmt `query:"upsert-push-subscription"`
	Delete     *sqlx.Stmt `query:"delete-push-subscription"`
	DeleteByID *sqlx.Stmt `query:"delete-push-subscription-by-id"`
}

type PushSubscription struct {
	ID       int    `db:"id" json:"id"`
	Endpoint string `db:"endpoint" json:"endpoint"`
	P256DH   string `db:"p256dh" json:"p256dh"`
	Auth     string `db:"auth" json:"auth"`
}

type PushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Tag   string `json:"tag,omitempty"`
	URL   string `json:"url"`
}

type pushDelivery struct {
	UserID  int
	Payload PushPayload
}

type pushSubscriptionStore interface {
	List(userID int) ([]PushSubscription, error)
	Upsert(userID int, subscription PushSubscriptionInput) error
	Delete(userID int, endpoint string) error
	DeleteByID(id int) error
}

type pushSettingStore interface {
	Get(key string) (types.JSONText, error)
	Update(settings any) error
}

type sqlPushStore struct {
	q pushQueries
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
		return nil, err
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

func (m *PushManager) Send(userID int, payload PushPayload) {
	select {
	case m.queue <- pushDelivery{UserID: userID, Payload: payload}:
	default:
		m.lo.Error("push notification queue is full", "user_id", userID)
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
				return err
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			m.lo.Error("push endpoint returned error", "subscription_id", subscription.ID, "status", resp.StatusCode)
		}
	}
	return nil
}

func (s *sqlPushStore) List(userID int) ([]PushSubscription, error) {
	var subscriptions []PushSubscription
	if err := s.q.List.Select(&subscriptions, userID); err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (s *sqlPushStore) Upsert(userID int, subscription PushSubscriptionInput) error {
	_, err := s.q.Upsert.Exec(userID, subscription.Endpoint, subscription.P256DH, subscription.Auth)
	return err
}

func (s *sqlPushStore) Delete(userID int, endpoint string) error {
	_, err := s.q.Delete.Exec(userID, endpoint)
	return err
}

func (s *sqlPushStore) DeleteByID(id int) error {
	_, err := s.q.DeleteByID.Exec(id)
	return err
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

func pushRoute(notificationType, conversationUUID, messageUUID string) string {
	list := "assigned"
	if notificationType == "mention" {
		list = "mentioned"
	}
	target := fmt.Sprintf("/inboxes/%s/conversation/%s", list, conversationUUID)
	if messageUUID != "" {
		target += "?scrollTo=" + messageUUID
	}
	return target
}

func newPushHTTPClient(lo *logf.Logger) *http.Client {
	control := ssrf.NewControl(true, nil, lo)
	return &http.Client{
		Transport: ssrf.NewTransport(control, 10*time.Second),
		Timeout:   10 * time.Second,
	}
}
