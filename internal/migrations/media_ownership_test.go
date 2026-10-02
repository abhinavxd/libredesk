package migrations

import (
	"github.com/jakedolan443/fernmail/internal/testutil"
	"testing"
)

func TestMediaOwnershipUpgradeRetainsLegacyDraftsWithoutInventingOwners(t *testing.T) {
	db := testutil.NewDB(t, "media_ownership_upgrade")
	db.MustExec(`
 DROP TABLE conversation_draft_media;
 ALTER TABLE media DROP COLUMN uploaded_by;
 INSERT INTO users(type,email,first_name) VALUES('agent','draft@test','Draft');
 INSERT INTO inboxes(name,channel) VALUES('Mail','email');
 INSERT INTO conversations(contact_id,inbox_id,status_id) SELECT u.id,i.id,s.id FROM users u CROSS JOIN inboxes i CROSS JOIN conversation_statuses s WHERE u.email='draft@test' AND s.name='Open';
 INSERT INTO media(store,filename,content_type,model_type) VALUES ('fs','inline.png','image/png','messages'),('fs','attached.txt','text/plain','messages'),('fs','orphan.txt','text/plain','messages');
 INSERT INTO conversation_drafts(conversation_id,user_id,type,content,meta)
 SELECT c.id,c.contact_id,'reply','<img src="cid:ldsk-' || m.uuid::text || '">',
  jsonb_build_object('attachments',jsonb_build_array(jsonb_build_object('id',(SELECT id FROM media WHERE filename='attached.txt'))))
 FROM conversations c CROSS JOIN media m WHERE m.filename='inline.png';
 INSERT INTO conversation_drafts(conversation_id,user_id,type,content,meta)
 SELECT id,contact_id,'private_note','Legacy metadata','{"attachments":"invalid-old-shape"}'::jsonb FROM conversations;
 `)
	for range 2 {
		if _, err := db.Exec(mediaOwnershipSQL); err != nil {
			t.Fatal(err)
		}
	}
	var retained, owned int
	if err := db.Get(&retained, `SELECT count(*) FROM conversation_draft_media`); err != nil || retained != 2 {
		t.Fatalf("retained=%d err=%v", retained, err)
	}
	if err := db.Get(&owned, `SELECT count(*) FROM media WHERE uploaded_by IS NOT NULL`); err != nil || owned != 0 {
		t.Fatalf("ownership inferred from unsafe metadata: %d %v", owned, err)
	}
	var views bool
	if err := db.Get(&views, `SELECT to_regclass('public.views') IS NOT NULL`); err != nil || views {
		t.Fatalf("Views resurrected: %v %v", views, err)
	}
}
