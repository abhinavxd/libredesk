package main

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat/proactive"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

type campaignEventRequest struct {
	DeliveryID string `json:"delivery_id"`
	BrowserKey string `json:"browser_key"`
	Event      string `json:"event"`
}

type campaignCandidate struct {
	campaign proactive.Campaign
	snapshot proactive.Snapshot
}

func handleWidgetCampaign(r *fastglue.Request) error {
	app := r.Context.(*App)
	var ctx proactive.Context
	if err := r.Decode(&ctx, "json"); err != nil || !validCampaignKey(ctx.BrowserKey) || !validCampaignKey(ctx.SessionKey) || len(ctx.URL) > 4096 || ctx.ActiveSeconds < 0 || ctx.ActiveSeconds > 86400 {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	ctx.Now = time.Now()
	ctx.Visitor = getWidgetIsVisitor(r)
	contact, err := widgetContact(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	ctx.ContactID = contact.ID
	inbox, err := getWidgetInbox(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	config, err := getWidgetConfig(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := checkConversationPermissions(app, config, ctx.Visitor, contact.ID, inbox.ID); err != nil {
		return r.SendEnvelope(nil)
	}
	loc, err := time.LoadLocation(app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	var candidates []campaignCandidate
	for _, campaign := range config.Campaigns {
		var hoursErr error
		within := func() bool {
			ok, err := campaignWithinHours(app, campaign, ctx.Now, loc)
			hoursErr = err
			return ok
		}
		reason := campaign.IneligibleReason(ctx, app.automation.MatchesContact(campaign.Conditions, contact), within)
		if hoursErr != nil {
			app.lo.Warn("skipping campaign with invalid business hours", "campaign_id", campaign.ID, "business_hours_id", campaign.BusinessHoursID)
			continue
		}
		if reason != "" {
			continue
		}
		sender, err := campaignSender(app, campaign.SenderID)
		if err != nil {
			continue
		}
		snapshot := proactive.Snapshot{Message: campaign.Message, Sender: strings.TrimSpace(sender.FullName()), Avatar: sender.AvatarURL.String, SenderID: sender.ID, TeamID: campaign.TeamID}
		if campaign.SenderID == 0 {
			snapshot.Sender = config.BrandName
		}
		candidates = append(candidates, campaignCandidate{campaign, snapshot})
	}
	if len(candidates) == 0 {
		return r.SendEnvelope(nil)
	}
	unlock := app.proactive.Lock()
	defer unlock()
	history, err := app.proactive.History(inbox.ID, ctx)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for _, c := range candidates {
		if proactive.Suppression(c.campaign, ctx, history, config.CampaignCooldownHours) != "" {
			continue
		}
		delivery, err := app.proactive.Reserve(inbox.ID, c.campaign, ctx, c.snapshot)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(delivery)
	}
	return r.SendEnvelope(nil)
}

func handleWidgetCampaignEvent(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req campaignEventRequest
	if err := r.Decode(&req, "json"); err != nil || !validCampaignKey(req.DeliveryID) || !validCampaignKey(req.BrowserKey) || !slices.Contains([]string{"displayed", "opened", "dismissed"}, req.Event) {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	inbox, err := getWidgetInbox(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	contactID, _ := getWidgetContactID(r)
	unlock := app.proactive.Lock()
	defer unlock()
	delivery, err := app.proactive.Get(req.DeliveryID, inbox.ID, req.BrowserKey, contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if req.Event == "displayed" {
		config, err := getWidgetConfig(r)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		active := slices.ContainsFunc(config.Campaigns, func(c proactive.Campaign) bool {
			return c.ID == delivery.CampaignID && c.Enabled
		})
		if !active || delivery.Dismissed || delivery.Replied || time.Since(delivery.CreatedAt) > time.Minute {
			return sendErrorEnvelope(r, envelope.NewError(envelope.ConflictError, app.i18n.T("widget.invitationExpired"), nil))
		}
	}
	if err := app.proactive.RecordEvent(delivery.ID, req.Event); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleCampaignStats(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	loc, err := time.LoadLocation(app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	from, err := time.ParseInLocation(time.DateOnly, string(r.RequestCtx.QueryArgs().Peek("from")), loc)
	if err != nil {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	to, err := time.ParseInLocation(time.DateOnly, string(r.RequestCtx.QueryArgs().Peek("to")), loc)
	if err != nil || to.Before(from) {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	stats, err := app.proactive.Stats(id, from, to.AddDate(0, 0, 1))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

func campaignWithinHours(app *App, c proactive.Campaign, now time.Time, loc *time.Location) (bool, error) {
	if c.BusinessHours == "any" {
		return true, nil
	}
	hours, err := app.businessHours.Get(c.BusinessHoursID)
	if err != nil {
		return false, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
	}
	if hours.IsAlwaysOpen {
		return true, nil
	}
	local := now.In(loc)
	var holidays []bhmodels.Holiday
	var working map[string]bhmodels.WorkingHours
	if err := json.Unmarshal(hours.Holidays, &holidays); err != nil {
		return false, campaignInputError(app)
	}
	if err := json.Unmarshal(hours.Hours, &working); err != nil {
		return false, campaignInputError(app)
	}
	for _, holiday := range holidays {
		if holiday.Date == local.Format(time.DateOnly) {
			return false, nil
		}
	}
	day, ok := working[local.Weekday().String()]
	if !ok {
		return false, nil
	}
	clock := local.Format("15:04")
	return withinWorkingHours(clock, day), nil
}

func withinWorkingHours(clock string, day bhmodels.WorkingHours) bool {
	if day.Open == day.Close {
		return false
	}
	if day.Open < day.Close {
		return clock >= day.Open && clock < day.Close
	}
	return clock >= day.Open || clock < day.Close
}

func campaignSender(app *App, id int) (umodels.User, error) {
	if id == 0 {
		return app.user.GetSystemUser()
	}
	return app.user.GetAgent(id, "")
}

func earliestCampaignDelay(campaigns []proactive.Campaign) int {
	delay := 0
	for _, campaign := range campaigns {
		if !campaign.Enabled {
			continue
		}
		if delay == 0 || campaign.DelaySeconds < delay {
			delay = campaign.DelaySeconds
		}
	}
	return delay
}

func validCampaignKey(key string) bool {
	id, err := uuid.Parse(key)
	return err == nil && id != uuid.Nil
}

func campaignInputError(app *App) error {
	return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
}

func validateWidgetFeatures(app *App, config livechat.Config) error {
	if config.Help.HelpCenterID < 0 || len(config.Help.FeaturedIDs) > 10 || config.Previews.AutoHideSeconds < 0 || config.Previews.AutoHideSeconds > 300 || !slices.Contains([]string{"", "message", "generic"}, config.Previews.Content) || len(config.Campaigns) > 50 || config.CampaignCooldownHours < 0 || config.CampaignCooldownHours > 8760 {
		return campaignInputError(app)
	}
	if config.Help.HelpCenterID > 0 {
		if _, err := app.helpcenter.GetHelpCenterByID(config.Help.HelpCenterID); err != nil {
			return err
		}
	}
	loc, err := time.LoadLocation(app.setting.GetAppTimezone())
	if err != nil {
		return campaignInputError(app)
	}
	ids := make(map[string]bool)
	for _, campaign := range config.Campaigns {
		if err := campaign.Validate(); err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("admin.inbox.livechat.campaignInvalid", "name", campaign.Name, "field", err.Error()), nil)
		}
		if ids[campaign.ID] {
			return campaignInputError(app)
		}
		ids[campaign.ID] = true
		if _, err := campaignSender(app, campaign.SenderID); err != nil {
			return err
		}
		if campaign.TeamID > 0 {
			if _, err := app.team.Get(campaign.TeamID); err != nil {
				return err
			}
		}
		if _, err := campaignWithinHours(app, campaign, time.Now(), loc); err != nil {
			return err
		}
	}
	return nil
}
