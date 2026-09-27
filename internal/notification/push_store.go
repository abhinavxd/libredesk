package notifier

import "github.com/jmoiron/sqlx"

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

type pushSubscriptionStore interface {
	List(userID int) ([]PushSubscription, error)
	Upsert(userID int, subscription PushSubscriptionInput) error
	Delete(userID int, endpoint string) error
	DeleteByID(id int) error
}

type sqlPushStore struct {
	q pushQueries
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
