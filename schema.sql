CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP TYPE IF EXISTS "channels" CASCADE; CREATE TYPE "channels" AS ENUM ('email', 'livechat');
DROP TYPE IF EXISTS "message_type" CASCADE; CREATE TYPE "message_type" AS ENUM ('incoming','outgoing','activity');
DROP TYPE IF EXISTS "message_sender_type" CASCADE; CREATE TYPE "message_sender_type" AS ENUM ('agent','contact');
DROP TYPE IF EXISTS "message_status" CASCADE; CREATE TYPE "message_status" AS ENUM ('received','sent','failed','pending');
DROP TYPE IF EXISTS "content_type" CASCADE; CREATE TYPE "content_type" AS ENUM ('text','html');
DROP TYPE IF EXISTS "conversation_assignment_type" CASCADE; CREATE TYPE "conversation_assignment_type" AS ENUM ('Round robin','Manual');
DROP TYPE IF EXISTS "template_type" CASCADE; CREATE TYPE "template_type" AS ENUM ('email_outgoing');
-- Visitors are unauthenticated contacts.
DROP TYPE IF EXISTS "user_type" CASCADE; CREATE TYPE "user_type" AS ENUM ('agent', 'contact', 'visitor', 'ai_assistant');
DROP TYPE IF EXISTS "media_disposition" CASCADE; CREATE TYPE "media_disposition" AS ENUM ('inline', 'attachment');
DROP TYPE IF EXISTS "media_store" CASCADE; CREATE TYPE "media_store" AS ENUM ('s3', 'fs');
DROP TYPE IF EXISTS "user_availability_status" CASCADE; CREATE TYPE "user_availability_status" AS ENUM ('online', 'away', 'away_manual', 'offline', 'away_and_reassigning');
DROP TYPE IF EXISTS "conversation_status_category" CASCADE; CREATE TYPE "conversation_status_category" AS ENUM ('open', 'waiting', 'resolved');
DROP TYPE IF EXISTS "webhook_event" CASCADE; CREATE TYPE webhook_event AS ENUM (
	'conversation.created',
	'conversation.status_changed',
	'conversation.assigned',
	'conversation.unassigned',
	'message.created',
	'message.updated'
);

-- Sequence to generate reference number for conversations.
DROP SEQUENCE IF EXISTS conversation_reference_number_sequence; CREATE SEQUENCE conversation_reference_number_sequence START 100;

-- Function to generate reference number for conversations with optional prefix.
CREATE OR REPLACE FUNCTION generate_reference_number(prefix TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN prefix || nextval('conversation_reference_number_sequence');
END;
$$ LANGUAGE plpgsql;

-- Function to pick the text search configuration for a help article locale.
CREATE OR REPLACE FUNCTION help_article_search_config(locale TEXT)
RETURNS regconfig AS $$
    SELECT CASE split_part(locale, '-', 1)
        WHEN 'ar' THEN 'arabic'
        WHEN 'da' THEN 'danish'
        WHEN 'nl' THEN 'dutch'
        WHEN 'en' THEN 'english'
        WHEN 'fi' THEN 'finnish'
        WHEN 'fr' THEN 'french'
        WHEN 'de' THEN 'german'
        WHEN 'el' THEN 'greek'
        WHEN 'hu' THEN 'hungarian'
        WHEN 'id' THEN 'indonesian'
        WHEN 'ga' THEN 'irish'
        WHEN 'it' THEN 'italian'
        WHEN 'lt' THEN 'lithuanian'
        WHEN 'ne' THEN 'nepali'
        WHEN 'no' THEN 'norwegian'
        WHEN 'pt' THEN 'portuguese'
        WHEN 'ro' THEN 'romanian'
        WHEN 'ru' THEN 'russian'
        WHEN 'es' THEN 'spanish'
        WHEN 'sv' THEN 'swedish'
        WHEN 'ta' THEN 'tamil'
        WHEN 'tr' THEN 'turkish'
        ELSE 'simple'
    END::regconfig;
$$ LANGUAGE sql IMMUTABLE;

DROP TABLE IF EXISTS inboxes CASCADE;
CREATE TABLE inboxes (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"uuid" UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE,
	"name" TEXT NOT NULL,
	deleted_at TIMESTAMPTZ NULL,
	channel channels NOT NULL,
	enabled bool DEFAULT TRUE NOT NULL,
	csat_enabled bool DEFAULT false NOT NULL,
	config jsonb DEFAULT '{}'::jsonb NOT NULL,
	"from" TEXT NULL,
	from_name_template TEXT NOT NULL DEFAULT '',
	secret TEXT NULL,
	linked_email_inbox_id INT REFERENCES inboxes(id) ON DELETE SET NULL,
	CONSTRAINT constraint_inboxes_on_name CHECK (length("name") <= 140)
);

DROP TABLE IF EXISTS teams CASCADE;
CREATE TABLE teams (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	emoji TEXT NULL,
	conversation_assignment_type conversation_assignment_type NOT NULL,
	max_auto_assigned_conversations INT DEFAULT 0 NOT NULL,

	-- Set to NULL when business hours or SLA policy is deleted.
	business_hours_id INT NULL,
	sla_policy_id INT NULL,

	timezone TEXT NULL,
	CONSTRAINT constraint_teams_on_emoji CHECK (length(emoji) <= 50),
	CONSTRAINT constraint_teams_on_name CHECK (length("name") <= 140),
	CONSTRAINT constraint_teams_on_timezone CHECK (length(timezone) <= 140),
	CONSTRAINT constraint_teams_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS roles CASCADE;
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    permissions TEXT[] DEFAULT '{}'::TEXT[] NOT NULL,
    "name" TEXT UNIQUE NOT NULL,
    description TEXT NULL,
	CONSTRAINT constraint_roles_on_name CHECK (length("name") <= 50),
	CONSTRAINT constraint_roles_on_description CHECK (length(description) <= 300)
);

DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    type user_type NOT NULL,
    deleted_at TIMESTAMPTZ NULL,
    enabled BOOL DEFAULT TRUE NOT NULL,
    email TEXT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NULL,
	phone_number_country_code TEXT NULL,
    phone_number TEXT NULL,
    country TEXT NULL,
    "password" VARCHAR(150) NULL,
    avatar_url TEXT NULL,
	custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL,
	external_user_id TEXT NULL,
    reset_password_token TEXT NULL,
    reset_password_token_expiry TIMESTAMPTZ NULL,
	availability_status user_availability_status DEFAULT 'offline' NOT NULL,
	last_active_at TIMESTAMPTZ NULL,
	last_login_at TIMESTAMPTZ NULL,
	-- API key authentication fields
	api_key TEXT NULL,
	api_secret TEXT NULL,
	api_key_last_used_at TIMESTAMPTZ NULL,
    CONSTRAINT constraint_users_on_country CHECK (LENGTH(country) <= 140),
    CONSTRAINT constraint_users_on_phone_number CHECK (LENGTH(phone_number) <= 20),
	CONSTRAINT constraint_users_on_phone_number_country_code CHECK (LENGTH(phone_number_country_code) <= 10),
    CONSTRAINT constraint_users_on_email_length CHECK (LENGTH(email) <= 320),
    CONSTRAINT constraint_users_on_first_name CHECK (LENGTH(first_name) <= 140),
    CONSTRAINT constraint_users_on_last_name CHECK (LENGTH(last_name) <= 140)
);
CREATE INDEX index_tgrm_users_on_email ON users USING GIN (email gin_trgm_ops);
CREATE INDEX index_users_on_api_key ON users(api_key);
CREATE INDEX index_users_on_availability_status_when_agent ON users(availability_status) WHERE type = 'agent' AND deleted_at IS NULL;
CREATE UNIQUE INDEX index_unique_users_on_email_when_type_is_agent
	ON users(email)
	WHERE type = 'agent' AND deleted_at IS NULL;
CREATE UNIQUE INDEX index_unique_users_on_ext_id_when_type_is_contact
	ON users (external_user_id)
	WHERE type = 'contact' AND deleted_at IS NULL AND external_user_id IS NOT NULL;
CREATE UNIQUE INDEX index_unique_users_on_email_when_no_ext_id_contact
	ON users (email)
	WHERE type = 'contact' AND deleted_at IS NULL AND external_user_id IS NULL;

DROP TABLE IF EXISTS user_image_permissions CASCADE;
CREATE TABLE user_image_permissions (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    senders TEXT[] NOT NULL DEFAULT '{}'::text[],
    CHECK (cardinality(senders) <= 100)
);

DROP TABLE IF EXISTS user_roles CASCADE;
CREATE TABLE user_roles (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),

	-- Cascade deletes when user or role is deleted, as they are not useful without each other.
	user_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	role_id INT REFERENCES roles(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,

	CONSTRAINT constraint_user_roles_on_user_id_and_role_id_unique UNIQUE (user_id, role_id)
);
CREATE INDEX index_user_roles_on_user_id ON user_roles(user_id);

DROP TABLE IF EXISTS inbox_access, inbox_users, inbox_roles CASCADE;

-- Existing inboxes remain available until an administrator restricts them.
CREATE TABLE IF NOT EXISTS inbox_access (
    inbox_id INTEGER PRIMARY KEY REFERENCES inboxes(id) ON DELETE CASCADE,
    restricted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS inbox_users (
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (inbox_id, user_id)
);
CREATE TABLE IF NOT EXISTS inbox_roles (
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (inbox_id, role_id)
);

-- One policy for HTTP reads, lists, search, counts and live notifications.
CREATE OR REPLACE FUNCTION can_access_inbox(target_inbox INTEGER, viewer BIGINT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM inboxes i
        JOIN users u ON u.id = viewer AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
        WHERE i.id = target_inbox AND i.channel = 'email' AND i.deleted_at IS NULL
        AND (
            u.email = 'System'
            OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                       WHERE ur.user_id = viewer AND r.name = 'Admin')
            OR EXISTS (SELECT 1 FROM inbox_users iu WHERE iu.inbox_id = i.id AND iu.user_id = viewer)
            OR EXISTS (SELECT 1 FROM inbox_roles ir JOIN user_roles ur ON ur.role_id = ir.role_id
                       WHERE ir.inbox_id = i.id AND ur.user_id = viewer)
        )
    );
$$;

-- Addresses are the user-facing mail endpoints. A mailbox owns an IMAP/SMTP
-- transport inbox; an alias deliberately shares one. Keeping the endpoint
-- separate from the transport lets permissions, routing, and reply identity
-- agree without polling an alias more than once.
DROP TABLE IF EXISTS email_address_teams, email_address_users, email_addresses CASCADE;
CREATE TABLE email_addresses (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'alias' CHECK (kind IN ('mailbox', 'alias')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT constraint_email_addresses_address CHECK (length(address) <= 320),
    CONSTRAINT constraint_email_addresses_display_name CHECK (length(display_name) <= 140)
);
CREATE UNIQUE INDEX index_email_addresses_on_normalized_address ON email_addresses (lower(address));
CREATE UNIQUE INDEX index_email_addresses_one_mailbox_per_inbox
    ON email_addresses (inbox_id) WHERE kind = 'mailbox';
CREATE INDEX index_email_addresses_on_inbox_id ON email_addresses(inbox_id);

CREATE TABLE email_address_users (
    address_id INTEGER NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (address_id, user_id)
);
CREATE TABLE email_address_teams (
    address_id INTEGER NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
    team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (address_id, team_id)
);
CREATE INDEX index_email_address_users_on_user_id ON email_address_users(user_id);
CREATE INDEX index_email_address_teams_on_team_id ON email_address_teams(team_id);

DROP TABLE IF EXISTS team_members CASCADE;
CREATE TABLE team_members (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	-- Cascade deletes when team or user is deleted.
	team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	emoji TEXT NULL,
	CONSTRAINT constraint_team_members_on_emoji CHECK (length(emoji) <= 1)
);
CREATE UNIQUE INDEX index_unique_team_members_on_team_id_and_user_id ON team_members (team_id, user_id);
CREATE INDEX index_team_members_on_user_id ON team_members (user_id);

-- Address policies are the sole user-facing access control. The IMAP/SMTP
-- transport is implementation detail, so a hidden transport policy can never
-- override an explicit Address grant. Admins see every address; everyone else
-- needs a direct or team grant, there is no open-to-all address.
CREATE OR REPLACE FUNCTION can_access_email_address(target_address INTEGER, viewer BIGINT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
    SELECT EXISTS (
        SELECT 1
        FROM email_addresses a
        JOIN inboxes i ON i.id = a.inbox_id
            AND i.channel = 'email'
            AND i.deleted_at IS NULL
        JOIN users u ON u.id = viewer AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
        WHERE a.id = target_address
          AND (
              u.email = 'System'
              OR EXISTS (
                  SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                  WHERE ur.user_id = viewer AND r.name = 'Admin'
              )
              OR EXISTS (
                  SELECT 1 FROM email_address_users eau
                  WHERE eau.address_id = a.id AND eau.user_id = viewer
              )
              OR EXISTS (
                  SELECT 1
                  FROM email_address_teams eat
                  JOIN team_members tm ON tm.team_id = eat.team_id
                  WHERE eat.address_id = a.id AND tm.user_id = viewer
              )
          )
    );
$$;

DROP TABLE IF EXISTS conversation_statuses CASCADE;
CREATE TABLE conversation_statuses (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL UNIQUE,
	category conversation_status_category NOT NULL DEFAULT 'open'
);

DROP TABLE IF EXISTS conversations CASCADE;
CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "uuid" UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE,
	reference_number TEXT DEFAULT generate_reference_number('') NOT NULL UNIQUE,

	-- Cascade deletes when contact is deleted.
    contact_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,

	-- Set to NULL when assigned user or team is deleted.
    assigned_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    assigned_team_id INT REFERENCES teams(id) ON DELETE SET NULL ON UPDATE CASCADE,

	-- Set to NULL when SLA policy is deleted.
	sla_policy_id INT NULL,

    -- Cascade deletes when inbox is deleted.
	inbox_id INT REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,

	-- Email addresses are the visible communication endpoints. This remains
	-- nullable for non-email channels.
	address_id INT REFERENCES email_addresses(id) ON DELETE SET NULL ON UPDATE CASCADE,

	-- Restrict delete.
	status_id INT REFERENCES conversation_statuses(id) ON DELETE RESTRICT ON UPDATE CASCADE NOT NULL,
    priority_id INT ,

	meta JSONB DEFAULT '{}'::jsonb NOT NULL,
	custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL,
	contact_last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    first_reply_at TIMESTAMPTZ NULL,
    last_reply_at TIMESTAMPTZ NULL,
    closed_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,

	"subject" TEXT NULL,
	waiting_since TIMESTAMPTZ NULL,
	last_message_at TIMESTAMPTZ NULL,
	last_message TEXT NULL,
	last_message_sender message_sender_type NULL,
	last_message_sender_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
	last_interaction TEXT NULL,
	last_interaction_sender message_sender_type NULL,
	last_interaction_sender_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
	last_interaction_at TIMESTAMPTZ NULL,
	next_sla_deadline_at TIMESTAMPTZ NULL,
	snoozed_until TIMESTAMPTZ NULL,
	last_continuity_email_sent_at TIMESTAMPTZ NULL
);
CREATE INDEX index_conversations_on_assigned_user_id ON conversations (assigned_user_id);
CREATE INDEX index_conversations_on_assigned_team_id ON conversations (assigned_team_id);
CREATE INDEX index_conversations_on_snoozed_until ON conversations (snoozed_until);
CREATE INDEX index_conversations_on_contact_id ON conversations (contact_id);
CREATE INDEX index_conversations_on_inbox_id ON conversations (inbox_id);
CREATE INDEX index_conversations_on_address_id ON conversations (address_id);
CREATE INDEX index_conversations_on_status_id ON conversations (status_id);
CREATE INDEX index_conversations_on_priority_id ON conversations (priority_id);
CREATE INDEX index_conversations_on_created_at ON conversations (created_at);
CREATE INDEX index_conversations_on_resolved_at ON conversations (resolved_at);
CREATE INDEX index_conversations_on_last_message_at ON conversations (last_message_at);
CREATE INDEX index_conversations_on_last_interaction_at ON conversations (last_interaction_at);
CREATE INDEX index_conversations_on_next_sla_deadline_at ON conversations (next_sla_deadline_at);
CREATE INDEX index_conversations_on_waiting_since ON conversations (waiting_since);
CREATE INDEX index_conversations_on_last_continuity_email_sent_at ON conversations (last_continuity_email_sent_at);

DROP TABLE IF EXISTS conversation_messages CASCADE;
CREATE TABLE conversation_messages (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "uuid" UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE,
    "type" message_type NOT NULL,
    status message_status NOT NULL,
    private BOOL DEFAULT FALSE NOT NULL,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    content_type content_type NULL,
    "content" TEXT NULL,
	text_content TEXT NULL,
    source_id TEXT NULL,
 	sender_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    sender_type message_sender_type NOT NULL,
    meta JSONB DEFAULT '{}'::JSONB NULL
);
CREATE INDEX index_conversation_messages_on_conversation_id ON conversation_messages (conversation_id);
CREATE INDEX index_conversation_messages_on_created_at ON conversation_messages (created_at);
CREATE INDEX index_conversation_messages_on_source_id ON conversation_messages (source_id);
CREATE INDEX index_conversation_messages_on_status ON conversation_messages (status);
CREATE INDEX index_conversation_messages_on_conversation_id_and_created_at ON conversation_messages (conversation_id, created_at);

DROP TABLE IF EXISTS conversation_drafts CASCADE;
CREATE TABLE conversation_drafts (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    type TEXT NOT NULL DEFAULT 'reply',
    content TEXT NOT NULL,
	meta JSONB DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT constraint_conversation_drafts_on_type CHECK (type IN ('reply', 'private_note'))
);
CREATE UNIQUE INDEX index_uniq_conversation_drafts_on_conversation_id_and_user_id_and_type ON conversation_drafts (conversation_id, user_id, type);

DROP TABLE IF EXISTS conversation_participants CASCADE;
CREATE TABLE conversation_participants (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	-- Cascade deletes when user or conversation is deleted.
	user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL
);
CREATE UNIQUE INDEX index_unique_conversation_participants_on_conversation_id_and_user_id ON conversation_participants (conversation_id, user_id);

DROP TABLE IF EXISTS conversation_mentions CASCADE;
CREATE TABLE conversation_mentions (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	mentioned_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	mentioned_team_id INT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
	mentioned_by_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	CONSTRAINT constraint_mention_target CHECK (
		(mentioned_user_id IS NOT NULL AND mentioned_team_id IS NULL) OR
		(mentioned_user_id IS NULL AND mentioned_team_id IS NOT NULL)
	)
);
CREATE INDEX index_conversation_mentions_on_mentioned_user_id ON conversation_mentions(mentioned_user_id);
CREATE INDEX index_conversation_mentions_on_mentioned_team_id ON conversation_mentions(mentioned_team_id);
CREATE INDEX index_conversation_mentions_on_conversation_id ON conversation_mentions(conversation_id);

DROP TABLE IF EXISTS conversation_last_seen CASCADE;
CREATE TABLE conversation_last_seen (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	last_seen_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE UNIQUE INDEX index_unique_conversation_last_seen ON conversation_last_seen (conversation_id, user_id);

DROP TABLE IF EXISTS media CASCADE;
CREATE TABLE media (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE,
	store "media_store" NOT NULL,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL,
	content_id TEXT NULL,
	model_id INT NULL,
	model_type TEXT NULL,
	disposition media_disposition NULL,
	"size" INT NULL,
	meta jsonb DEFAULT '{}'::jsonb NOT NULL,
	private BOOLEAN NOT NULL DEFAULT true,
	CONSTRAINT constraint_media_on_filename CHECK (length(filename) <= 1000),
	CONSTRAINT constraint_media_on_content_id CHECK (length(content_id) <= 300)
);
CREATE INDEX index_media_on_model_type_and_model_id ON media(model_type, model_id);
CREATE INDEX index_media_on_content_id ON media(content_id);

DROP TABLE IF EXISTS message_image_permissions;
CREATE TABLE message_image_permissions (
 user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
 message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE,
 content_hash TEXT NOT NULL CHECK (length(content_hash) = 64),
 PRIMARY KEY (user_id, message_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_media_resource_image_source
 ON media (model_id, content_id) WHERE model_type = 'resource_images';
CREATE UNIQUE INDEX IF NOT EXISTS index_media_resource_avatar_source
 ON media (model_id, content_id) WHERE model_type = 'resource_avatars';

DROP TABLE IF EXISTS resource_image_cache_pending;
CREATE TABLE resource_image_cache_pending (
 uuid UUID PRIMARY KEY,
 size BIGINT NOT NULL CHECK (size >= 0),
 state TEXT NOT NULL CHECK (state IN ('upload', 'delete')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS index_media_resource_cache_fifo
 ON media (created_at, id) INCLUDE (uuid, size)
 WHERE model_type IN ('resource_images', 'resource_avatars');

DROP TABLE IF EXISTS oidc CASCADE;
CREATE TABLE oidc (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NULL,
	provider_url TEXT NOT NULL,
	client_id TEXT NOT NULL,
	client_secret TEXT NOT NULL,
	enabled bool DEFAULT TRUE NOT NULL,
	provider VARCHAR NULL,
	logo_url TEXT NOT NULL DEFAULT '',
	CONSTRAINT constraint_oidc_on_name CHECK (length("name") <= 140)
);

DROP TABLE IF EXISTS settings CASCADE;
CREATE TABLE settings (
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"key" TEXT NOT NULL UNIQUE,
	value jsonb DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT settings_key_key UNIQUE ("key")
);
CREATE INDEX index_settings_on_key ON settings USING btree ("key");

DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	type template_type NOT NULL,
	body TEXT NOT NULL,
	is_default bool DEFAULT false NOT NULL,
	"name" TEXT NOT NULL,
	subject TEXT NULL,
	is_builtin bool DEFAULT false NOT NULL,
	CONSTRAINT constraint_templates_on_name CHECK (length("name") <= 140),
	CONSTRAINT constraint_templates_on_subject CHECK (length(subject) <= 1000)
);
CREATE UNIQUE INDEX index_unique_templates_on_is_default_when_is_default_is_true ON templates USING btree (is_default)
WHERE (is_default = true);

DROP TABLE IF EXISTS webhooks CASCADE;
CREATE TABLE webhooks (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	name TEXT NOT NULL,
	url TEXT NOT NULL,
	events webhook_event[] NOT NULL DEFAULT '{}',
	secret TEXT DEFAULT '',
	is_active BOOLEAN DEFAULT true,
	CONSTRAINT constraint_webhooks_on_name CHECK (length(name) <= 255),
	CONSTRAINT constraint_webhooks_on_url CHECK (length(url) <= 2048),
	CONSTRAINT constraint_webhooks_on_secret CHECK (length(secret) <= 255),
	CONSTRAINT constraint_webhooks_on_events_not_empty CHECK (array_length(events, 1) > 0)
);

-- Default AI prompts

-- Default settings
INSERT INTO settings ("key", value)
VALUES
    ('app.lang', '"en-US"'::jsonb),
    ('security.resource_policy', '{"mode":"load_on_receipt","allowed_domains":[]}'::jsonb),
    ('app.root_url', '"http://localhost:9000"'::jsonb),
    ('app.logo_url', '""'::jsonb),
    ('app.site_name', '""'::jsonb),
    ('app.max_file_upload_size', '20'::jsonb),
    ('app.allowed_file_upload_extensions', '["*"]'::jsonb),
	('app.timezone', '"Asia/Kolkata"'::jsonb),
	('app.show_conversation_subject', 'true'::jsonb),
    ('account_email.username', '"admin@yourcompany.com"'::jsonb),
    ('account_email.host', '""'::jsonb),
    ('account_email.port', '587'::jsonb),
    ('account_email.password', '""'::jsonb),
    ('account_email.max_conns', '5'::jsonb),
    ('account_email.idle_timeout', '"25s"'::jsonb),
    ('account_email.wait_timeout', '"60s"'::jsonb),
    ('account_email.auth_protocol', '"plain"'::jsonb),
	('account_email.tls_type', '"starttls"'::jsonb),
	('account_email.tls_skip_verify', 'false'::jsonb),
	('account_email.hello_hostname', '""'::jsonb),
    ('account_email.email_address', '"admin@yourcompany.com"'::jsonb),
    ('account_email.max_msg_retries', '3'::jsonb),
    ('account_email.enabled', 'false'::jsonb);

-- Default conversation priorities

-- Default conversation statuses
INSERT INTO conversation_statuses (name, category) VALUES
('Open', 'open'),
('Snoozed', 'waiting'),
('Resolved', 'resolved'),
('Closed', 'resolved');

-- Default roles
INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Agent',
		'Role for all agents with limited access to conversations.',
		'{conversations:read_all,conversations:read,conversations:create,messages:read,messages:write,messages:write_private,reviews:manage}'
	);

INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Contributor',
		'Reads assigned addresses; replies and new emails are reviewed before sending.',
		'{conversations:read_all,conversations:read,conversations:create,messages:read,reviews:submit}'
	);

INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Admin',
		'Role for users who have complete access to everything.',
		'{webhooks:manage,general_settings:manage,activation_keys:manage,oidc:manage,conversations:read_all,conversations:read,conversations:create,messages:read,messages:write,messages:write_private,reviews:manage,users:manage,inboxes:manage,templates:manage}'
	);

-- BEGIN mail_reliability schema
-- Durable mailbox progress and retries; a new UIDVALIDITY gets an independent full scan.
CREATE TABLE IF NOT EXISTS mail_sync_cursors (
 inbox_id integer NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
 mailbox_key text NOT NULL, uid_validity bigint NOT NULL, last_uid bigint NOT NULL DEFAULT 0,
 PRIMARY KEY(inbox_id,mailbox_key,uid_validity)
);
CREATE TABLE IF NOT EXISTS mail_sync_failures (
 inbox_id integer NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
 mailbox_key text NOT NULL, uid_validity bigint NOT NULL, uid bigint NOT NULL,
 error text NOT NULL, updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(inbox_id,mailbox_key,uid_validity,uid)
);
CREATE TABLE IF NOT EXISTS incoming_mail_queue (
 id bigserial PRIMARY KEY, inbox_id integer NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
 delivery_key text NOT NULL UNIQUE, payload jsonb, attempts integer NOT NULL DEFAULT 0,
 next_attempt_at timestamptz NOT NULL DEFAULT now(), lease_until timestamptz,
 claim_token uuid, last_error text, completed_at timestamptz
);
CREATE INDEX IF NOT EXISTS incoming_mail_queue_pending ON incoming_mail_queue(next_attempt_at) WHERE completed_at IS NULL;
CREATE INDEX IF NOT EXISTS incoming_mail_queue_inbox_pending ON incoming_mail_queue(inbox_id,id) WHERE completed_at IS NULL;
CREATE INDEX IF NOT EXISTS incoming_mail_queue_completed ON incoming_mail_queue(completed_at,id) WHERE completed_at IS NOT NULL;
-- Protect address-local Message-ID identity without deleting historical duplicates.
CREATE TABLE IF NOT EXISTS received_mail_sources (
 address_id integer NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
 source_id text NOT NULL, message_id bigint NOT NULL REFERENCES conversation_messages(id) ON DELETE CASCADE,
 PRIMARY KEY(address_id,source_id)
);
INSERT INTO received_mail_sources(address_id,source_id,message_id)
 SELECT c.address_id,m.source_id,min(m.id) FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id
 WHERE c.address_id IS NOT NULL AND m.type='incoming' AND m.source_id IS NOT NULL AND m.source_id<>''
 GROUP BY c.address_id,m.source_id ON CONFLICT DO NOTHING;
CREATE OR REPLACE FUNCTION reserve_received_mail_source() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE endpoint integer;
BEGIN
 IF NEW.type='incoming' AND NEW.source_id IS NOT NULL AND NEW.source_id<>'' THEN
  SELECT address_id INTO endpoint FROM conversations WHERE id=NEW.conversation_id;
  IF endpoint IS NOT NULL THEN
   INSERT INTO received_mail_sources(address_id,source_id,message_id) VALUES(endpoint,NEW.source_id,NEW.id);
  END IF;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS reserve_received_mail_source ON conversation_messages;
CREATE TRIGGER reserve_received_mail_source AFTER INSERT ON conversation_messages FOR EACH ROW EXECUTE FUNCTION reserve_received_mail_source();
CREATE TABLE IF NOT EXISTS mail_delivery_attempts (
 id bigserial PRIMARY KEY, message_id bigint NOT NULL REFERENCES conversation_messages(id) ON DELETE CASCADE,
 token uuid NOT NULL UNIQUE, state text NOT NULL CHECK(state IN ('sending','sent','failed','unknown','retry_authorized')),
 created_at timestamptz NOT NULL DEFAULT now(), lease_until timestamptz NOT NULL,
 finished_at timestamptz, error text
);
CREATE UNIQUE INDEX IF NOT EXISTS mail_delivery_attempts_active ON mail_delivery_attempts(message_id) WHERE state IN ('sending','unknown');
ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS reply_to_source_id text NOT NULL DEFAULT '';
-- END mail_reliability schema

-- BEGIN media_ownership schema
-- Browser uploads have an owner before any message or draft refers to them.
-- Existing unlinked files have no trustworthy uploader provenance; do not infer
-- ownership from user-controlled legacy draft metadata. Reupload those files.
ALTER TABLE media ADD COLUMN IF NOT EXISTS uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS index_media_on_uploaded_by ON media(uploaded_by) WHERE uploaded_by IS NOT NULL;

CREATE TABLE IF NOT EXISTS conversation_draft_media (
    draft_id BIGINT NOT NULL REFERENCES conversation_drafts(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    PRIMARY KEY (draft_id, media_id)
);
CREATE INDEX IF NOT EXISTS index_conversation_draft_media_on_media_id ON conversation_draft_media(media_id);

-- Protect existing draft references from collection without granting ownership.
-- Match strings instead of casting legacy metadata supplied by browsers.
INSERT INTO conversation_draft_media(draft_id, media_id)
SELECT DISTINCT d.id, m.id
FROM conversation_drafts d JOIN media m ON
    d.content LIKE '%cid:ldsk-' || m.uuid::text || '%'
    OR d.content LIKE '%/uploads/' || m.uuid::text || '%'
    OR EXISTS (
        SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(d.meta->'attachments')='array'
            THEN d.meta->'attachments' ELSE '[]'::jsonb END) attachment
        WHERE attachment->>'id'=m.id::text OR attachment->>'uuid'=m.uuid::text
    )
WHERE COALESCE(m.model_id,0)=0 AND (m.model_type='messages' OR m.model_type IS NULL)
ON CONFLICT DO NOTHING;
-- END media_ownership schema

-- BEGIN outbound_reviews schema
-- Contributor emails wait here until an Admin or Agent approves or denies them.
-- A submission is not a message: it is never dispatched, searched or counted
-- as unread. Approval queues a real message in the same transaction.
CREATE TABLE IF NOT EXISTS outbound_reviews (
    id BIGSERIAL PRIMARY KEY,
    "uuid" UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    kind TEXT NOT NULL CHECK (kind IN ('reply', 'new')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'withdrawn')),
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address_id INTEGER NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE,
    subject TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    "to" TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    cc TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    bcc TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    reviewer_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    decision_note TEXT NOT NULL DEFAULT '',
    dismissed_at TIMESTAMPTZ,
    message_id BIGINT REFERENCES conversation_messages(id) ON DELETE SET NULL,
    CONSTRAINT constraint_outbound_reviews_reply_conversation CHECK (kind <> 'reply' OR conversation_id IS NOT NULL),
    CONSTRAINT constraint_outbound_reviews_subject CHECK (length(subject) <= 998),
    CONSTRAINT constraint_outbound_reviews_content CHECK (length(content) <= 1048576),
    CONSTRAINT constraint_outbound_reviews_note CHECK (length(decision_note) <= 2000)
);
CREATE INDEX IF NOT EXISTS index_outbound_reviews_on_status_and_address ON outbound_reviews(status, address_id);
CREATE INDEX IF NOT EXISTS index_outbound_reviews_on_author_and_status ON outbound_reviews(author_id, status);
CREATE INDEX IF NOT EXISTS index_outbound_reviews_on_conversation_id ON outbound_reviews(conversation_id);
-- One pending reply per contributor per conversation; the composer locks meanwhile.
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_outbound_reviews_pending_reply
    ON outbound_reviews(author_id, conversation_id) WHERE status = 'pending' AND kind = 'reply';

-- Uploads referenced by a submission are protected from cleanup like draft uploads.
CREATE TABLE IF NOT EXISTS outbound_review_media (
    review_id BIGINT NOT NULL REFERENCES outbound_reviews(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    inline BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (review_id, media_id)
);
CREATE INDEX IF NOT EXISTS index_outbound_review_media_on_media_id ON outbound_review_media(media_id);
-- END outbound_reviews schema

-- BEGIN activation_keys schema
-- Key Distribution: per-app pools of activation keys handed out from the
-- composer. A key leaves the redeemable pool only when an email carrying its
-- placeholder is queued, and it never returns.
CREATE TABLE IF NOT EXISTS activation_key_apps (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL,
    archived_at TIMESTAMPTZ,
    CONSTRAINT constraint_activation_key_apps_name CHECK (length(name) BETWEEN 1 AND 100)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_activation_key_apps_name ON activation_key_apps (lower(name));

CREATE TABLE IF NOT EXISTS activation_keys (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    app_id INTEGER NOT NULL REFERENCES activation_key_apps(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'redeemable' CHECK (status IN ('redeemable', 'activated', 'voided')),
    -- AES-GCM ciphertext. key_hash is an HMAC of the key, used to reject
    -- duplicates and to find a key without decrypting the pool.
    key_encrypted TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    added_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    activated_at TIMESTAMPTZ,
    message_id BIGINT REFERENCES conversation_messages(id) ON DELETE SET NULL,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE SET NULL,
    placeholder_id TEXT,
    recipients TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    sent_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    voided_at TIMESTAMPTZ,
    voided_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT constraint_activation_keys_activated CHECK (status <> 'activated' OR activated_at IS NOT NULL)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_activation_keys_hash ON activation_keys (key_hash) WHERE status <> 'voided';
CREATE INDEX IF NOT EXISTS index_activation_keys_pool ON activation_keys (app_id, status, id);
CREATE INDEX IF NOT EXISTS index_activation_keys_message ON activation_keys (message_id) WHERE message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activation_keys_conversation ON activation_keys (conversation_id) WHERE conversation_id IS NOT NULL;

-- Keys only move forward: redeemable to activated (by sending) or voided (by
-- an admin). Activated keys can't be edited, reset or deleted.
CREATE OR REPLACE FUNCTION guard_activation_key_status() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'activated' THEN
            RAISE EXCEPTION 'activated keys cannot be deleted';
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.status IS DISTINCT FROM OLD.status AND OLD.status <> 'redeemable' THEN
        RAISE EXCEPTION 'activation key status % is final', OLD.status;
    END IF;
    IF OLD.status = 'activated' AND (NEW.key_encrypted IS DISTINCT FROM OLD.key_encrypted
        OR NEW.key_hash IS DISTINCT FROM OLD.key_hash OR NEW.app_id IS DISTINCT FROM OLD.app_id) THEN
        RAISE EXCEPTION 'activated keys cannot be changed';
    END IF;
    RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS guard_activation_key_status ON activation_keys;
CREATE TRIGGER guard_activation_key_status BEFORE UPDATE OR DELETE ON activation_keys
    FOR EACH ROW EXECUTE FUNCTION guard_activation_key_status();

-- The app each person last picked in the composer's key card.
CREATE TABLE IF NOT EXISTS activation_key_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    app_id INTEGER REFERENCES activation_key_apps(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings ("key", value)
VALUES ('key_distribution', '{"enabled":false,"max_keys_per_email":1,"low_stock_threshold":10}'::jsonb)
ON CONFLICT ("key") DO NOTHING;

UPDATE roles SET permissions = array_append(permissions, 'activation_keys:manage')
WHERE name = 'Admin' AND NOT ('activation_keys:manage' = ANY(permissions));
-- END activation_keys schema
