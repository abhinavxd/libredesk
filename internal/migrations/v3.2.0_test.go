package migrations

import (
	"github.com/jakedolan443/fernmail/internal/testutil"
	"testing"
)

func TestRetireTagsPreservesRemainingViewFilters(t *testing.T) {
	db := testutil.NewDB(t, "retire_tags")
	db.MustExec(`
 CREATE TYPE view_visibility AS ENUM ('all', 'team', 'user');
 CREATE TABLE views(id SERIAL PRIMARY KEY, name TEXT NOT NULL, visibility view_visibility NOT NULL,
   filters JSONB NOT NULL DEFAULT '[]', user_id BIGINT, team_id INTEGER);
 CREATE TABLE tags(id int);
 CREATE TABLE conversation_tags(tag_id int);
 ALTER TABLE inboxes ADD COLUMN prompt_tags_on_reply bool;
 INSERT INTO inboxes(name,channel) VALUES ('Keep mail','email');
 INSERT INTO views(name,visibility,filters) VALUES
 ('Mixed','all','{"logic":"AND","rules":[{"field":"tags","model":"conversations","operator":"contains","value":"[1]"},{"logic":"OR","rules":[{"field":"tags","model":"conversations","value":"[2]"},{"field":"status_id","model":"conversations","operator":"equals","value":"1"}]}]}'),
 ('Tag only','all','[{"field":"tags","model":"conversations","value":"[1]"}]'),
 ('Inbox','all','[{"field":"inbox_id","model":"conversations","operator":"equals","value":"1"}]');
 `)
	for range 2 {
		if err := V3_2_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var ok bool
	for _, q := range []string{
		`SELECT filters = '{"logic":"AND","rules":[{"logic":"OR","rules":[{"field":"status_id","model":"conversations","operator":"equals","value":"1"}]}]}'::jsonb FROM views WHERE name='Mixed'`,
		`SELECT filters = '[]'::jsonb FROM views WHERE name='Tag only'`,
		`SELECT filters = '[{"field":"inbox_id","model":"conversations","operator":"equals","value":"1"}]'::jsonb FROM views WHERE name='Inbox'`,
		`SELECT count(*)=1 FROM inboxes WHERE name='Keep mail'`,
		`SELECT to_regclass('tags') IS NULL AND to_regclass('conversation_tags') IS NULL`,
	} {
		if err := db.Get(&ok, q); err != nil || !ok {
			t.Fatalf("%s: %v %v", q, ok, err)
		}
	}
}
