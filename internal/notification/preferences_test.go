package notifier

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestEnabledChannelsReturnsLookupFailure(t *testing.T) {
	db := testutil.NewDB(t, "notification_preferences_read_failure")
	lo := logf.New(logf.Opts{})
	manager, err := NewPreferenceManager(PreferenceManagerOpts{
		DB:   db,
		Lo:   &lo,
		I18n: testutil.NewI18n(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.q.GetPreferencesType.Close(); err != nil {
		t.Fatal(err)
	}

	channels, err := manager.EnabledChannels([]int{1}, models.NotificationTypeAssignment)
	if err == nil {
		t.Fatal("preference lookup failure was not returned")
	}
	if channels != nil {
		t.Fatalf("enabled channels = %v, want nil", channels)
	}
}

func TestPreferenceUpdateIsAtomic(t *testing.T) {
	db := testutil.NewDB(t, "notification_preferences_atomic")
	var userID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'prefs@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	manager, err := NewPreferenceManager(PreferenceManagerOpts{
		DB:   db,
		Lo:   &lo,
		I18n: testutil.NewI18n(t),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("invalid preference", func(t *testing.T) {
		err := manager.Update(userID, []models.NotificationPreference{
			{NotificationType: models.NotificationTypeAssignment, Channel: models.NotificationChannelEmail, Enabled: false},
			{NotificationType: "unknown", Channel: models.NotificationChannelPush, Enabled: true},
		})
		if err == nil {
			t.Fatal("invalid update succeeded")
		}
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM user_notification_preferences WHERE user_id = $1`, userID); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("stored preferences = %d, want 0", count)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		db.MustExec(`
			CREATE FUNCTION reject_push_preference() RETURNS trigger AS $$
			BEGIN
				IF NEW.channel = 'push' THEN
					RAISE EXCEPTION 'push rejected';
				END IF;
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			CREATE TRIGGER reject_push_preference
			BEFORE INSERT OR UPDATE ON user_notification_preferences
			FOR EACH ROW EXECUTE FUNCTION reject_push_preference();
		`)
		err := manager.Update(userID, []models.NotificationPreference{
			{NotificationType: models.NotificationTypeAssignment, Channel: models.NotificationChannelEmail, Enabled: false},
			{NotificationType: models.NotificationTypeAssignment, Channel: models.NotificationChannelPush, Enabled: true},
		})
		if err == nil {
			t.Fatal("update succeeded despite database failure")
		}
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM user_notification_preferences WHERE user_id = $1`, userID); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("stored preferences = %d, want 0", count)
		}
	})
}
