package twitter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type Twitter struct {
	id                   int
	name                 string
	config               Config
	configMu             sync.RWMutex
	provider             Provider
	lo                   *logf.Logger
	messageStore         inbox.MessageStore
	userStore            inbox.UserStore
	wg                   sync.WaitGroup
	tokenRefreshCallback TokenRefreshCallback
}

type Opts struct {
	ID                   int
	Name                 string
	Config               Config
	Provider             Provider
	Lo                   *logf.Logger
	TokenRefreshCallback TokenRefreshCallback
}

func New(store inbox.MessageStore, userStore inbox.UserStore, opts Opts) (*Twitter, error) {
	provider := opts.Provider
	var err error
	if provider == nil {
		provider, err = NewProvider(opts.Config)
		if err != nil {
			return nil, err
		}
	}
	if opts.Config.PollInterval == "" {
		opts.Config.PollInterval = "5m"
	}
	if opts.Config.DeliveryMode == "" {
		opts.Config.DeliveryMode = DeliveryModePolling
	}
	return &Twitter{
		id:                   opts.ID,
		name:                 opts.Name,
		config:               opts.Config,
		provider:             provider,
		lo:                   opts.Lo,
		messageStore:         store,
		userStore:            userStore,
		tokenRefreshCallback: opts.TokenRefreshCallback,
	}, nil
}

func (t *Twitter) Identifier() int {
	return t.id
}

func (t *Twitter) Receive(ctx context.Context) error {
	cfg := t.currentConfig()
	if cfg.DeliveryMode != DeliveryModePolling {
		<-ctx.Done()
		return nil
	}

	if cfg.IngestDMs {
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			t.pollDMs(ctx)
		}()
	}
	if cfg.IngestMentions {
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			t.pollMentions(ctx)
		}()
	}
	if !cfg.IngestDMs && !cfg.IngestMentions {
		<-ctx.Done()
		return nil
	}
	t.wg.Wait()
	return nil
}

func (t *Twitter) Send(message cmodels.OutboundMessage) error {
	meta, err := ParseMessageMeta(message.Meta)
	if err != nil {
		return fmt.Errorf("unmarshalling twitter message meta: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch meta.TwitterSource {
	case TwitterSourceDM:
		externalID, err := t.userStore.GetContactExternalID(message.MessageReceiverID)
		if err != nil {
			return err
		}
		recipientID := strings.TrimPrefix(externalID, "x:")
		if recipientID == "" || recipientID == externalID {
			return fmt.Errorf("contact %d does not have a twitter external user ID", message.MessageReceiverID)
		}
		_, err = t.provider.SendDM(ctx, recipientID, message.TextContent, nil)
		return err
	case TwitterSourceMention:
		inReplyToID := meta.TweetID
		if inReplyToID == "" {
			inReplyToID = meta.InReplyToStatusID
		}
		if inReplyToID == "" {
			return fmt.Errorf("twitter mention reply is missing tweet_id")
		}
		if len([]rune(message.TextContent)) > 280 {
			return fmt.Errorf("twitter public replies cannot exceed 280 characters")
		}
		_, err := t.provider.Reply(ctx, inReplyToID, message.TextContent, nil)
		return err
	default:
		return fmt.Errorf("unknown twitter source: %s", meta.TwitterSource)
	}
}

func (t *Twitter) Close() error {
	return nil
}

func (t *Twitter) Name() string {
	return t.name
}

func (t *Twitter) FromAddress() string {
	cfg := t.currentConfig()
	if cfg.ScreenName == "" {
		return ""
	}
	return "@" + strings.TrimPrefix(cfg.ScreenName, "@")
}

func (t *Twitter) FromNameTemplate() string {
	return ""
}

func (t *Twitter) ReplyToAddress() string {
	return ""
}

func (t *Twitter) Channel() string {
	return ChannelTwitter
}

func (t *Twitter) currentConfig() Config {
	t.configMu.RLock()
	defer t.configMu.RUnlock()
	return t.config
}

func (t *Twitter) updateCursor(mut func(*Config)) {
	t.configMu.Lock()
	mut(&t.config)
	cfg := t.config
	t.configMu.Unlock()
	if t.tokenRefreshCallback != nil {
		if err := t.tokenRefreshCallback(t.id, cfg); err != nil {
			t.lo.Error("error persisting twitter cursor", "inbox_id", t.id, "error", err)
		}
	}
}

func incomingContact(user XUser) cmodels.IncomingContact {
	first, last := splitName(user.Name)
	return cmodels.IncomingContact{
		FirstName:      first,
		LastName:       last,
		ExternalUserID: null.StringFrom("x:" + user.ID),
		AvatarURL:      null.NewString(user.AvatarURL, user.AvatarURL != ""),
	}
}

func splitName(name string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

var _ inbox.Inbox = (*Twitter)(nil)
