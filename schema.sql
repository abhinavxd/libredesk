CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP TYPE IF EXISTS "channels" CASCADE; CREATE TYPE "channels" AS ENUM ('email');
DROP TYPE IF EXISTS "message_type" CASCADE; CREATE TYPE "message_type" AS ENUM ('incoming','outgoing','activity');
DROP TYPE IF EXISTS "message_sender_type" CASCADE; CREATE TYPE "message_sender_type" AS ENUM ('agent','contact');
DROP TYPE IF EXISTS "message_status" CASCADE; CREATE TYPE "message_status" AS ENUM ('received','sent','failed','pending');
DROP TYPE IF EXISTS "content_type" CASCADE; CREATE TYPE "content_type" AS ENUM ('text','html');
DROP TYPE IF EXISTS "conversation_assignment_type" CASCADE; CREATE TYPE "conversation_assignment_type" AS ENUM ('Round robin','Manual');
DROP TYPE IF EXISTS "template_type" CASCADE; CREATE TYPE "template_type" AS ENUM ('email_outgoing', 'email_notification');
DROP TYPE IF EXISTS "user_type" CASCADE; CREATE TYPE "user_type" AS ENUM ('agent', 'contact');
DROP TYPE IF EXISTS "ai_provider" CASCADE; CREATE TYPE "ai_provider" AS ENUM ('openai');
DROP TYPE IF EXISTS "automation_execution_mode" CASCADE; CREATE TYPE "automation_execution_mode" AS ENUM ('all', 'first_match');
DROP TYPE IF EXISTS "macro_visibility" CASCADE; CREATE TYPE "macro_visibility" AS ENUM ('all', 'team', 'user');
DROP TYPE IF EXISTS "media_disposition" CASCADE; CREATE TYPE "media_disposition" AS ENUM ('inline', 'attachment');
DROP TYPE IF EXISTS "media_store" CASCADE; CREATE TYPE "media_store" AS ENUM ('s3', 'fs');
DROP TYPE IF EXISTS "user_availability_status" CASCADE; CREATE TYPE "user_availability_status" AS ENUM ('online', 'away', 'away_manual', 'offline', 'away_and_reassigning');
DROP TYPE IF EXISTS "applied_sla_status" CASCADE; CREATE TYPE "applied_sla_status" AS ENUM ('pending', 'breached', 'met', 'partially_met');
DROP TYPE IF EXISTS "sla_event_status" CASCADE; CREATE TYPE "sla_event_status" AS ENUM ('pending', 'breached', 'met');
DROP TYPE IF EXISTS "sla_metric" CASCADE; CREATE TYPE "sla_metric" AS ENUM ('first_response', 'resolution', 'next_response');
DROP TYPE IF EXISTS "sla_notification_type" CASCADE; CREATE TYPE "sla_notification_type" AS ENUM ('warning', 'breach');
DROP TYPE IF EXISTS "activity_log_type" CASCADE; CREATE TYPE "activity_log_type" AS ENUM ('agent_login', 'agent_logout', 'agent_away', 'agent_away_reassigned', 'agent_online');
DROP TYPE IF EXISTS "macro_visible_when" CASCADE; CREATE TYPE "macro_visible_when" AS ENUM ('replying', 'starting_conversation', 'adding_private_note');
DROP TYPE IF EXISTS "webhook_event" CASCADE; CREATE TYPE webhook_event AS ENUM (
	'conversation.created',
	'conversation.status_changed',
	'conversation.tags_changed',
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

DROP TABLE IF EXISTS sla_policies CASCADE;
CREATE TABLE sla_policies (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	name TEXT NOT NULL,
	description TEXT NULL,
	first_response_time TEXT NOT NULL,
	resolution_time TEXT NOT NULL,
	next_response_time TEXT NULL,
	notifications JSONB DEFAULT '[]'::jsonb NOT NULL,
	CONSTRAINT constraint_sla_policies_on_name CHECK (length(name) <= 140),
	CONSTRAINT constraint_sla_policies_on_description CHECK (length(description) <= 300)
);

DROP TABLE IF EXISTS business_hours CASCADE;
CREATE TABLE business_hours (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
	name TEXT NOT NULL,
	description TEXT NULL,
	is_always_open BOOL DEFAULT false NOT NULL,
	hours JSONB NOT NULL,
	holidays JSONB DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT constraint_business_hours_on_name CHECK (length(name) <= 140),
	CONSTRAINT constraint_business_hours_on_description CHECK (length(description) <= 300)
);

DROP TABLE IF EXISTS inboxes CASCADE;
CREATE TABLE inboxes (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	deleted_at TIMESTAMPTZ NULL,
	channel channels NOT NULL,
	enabled bool DEFAULT TRUE NOT NULL,
	csat_enabled bool DEFAULT false NOT NULL,
	config jsonb DEFAULT '{}'::jsonb NOT NULL,
	"from" TEXT NULL,
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
	business_hours_id INT REFERENCES business_hours(id) ON DELETE SET NULL ON UPDATE CASCADE NULL,
	sla_policy_id INT REFERENCES sla_policies(id) ON DELETE SET NULL ON UPDATE CASCADE NULL,

	timezone TEXT NULL,
	CONSTRAINT constraint_teams_on_emoji CHECK (length(emoji) <= 10),
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
	phone_number_calling_code TEXT NULL,
    phone_number TEXT NULL,
    country TEXT NULL,
    "password" VARCHAR(150) NULL,
    avatar_url TEXT NULL,
	custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL,
    reset_password_token TEXT NULL,
    reset_password_token_expiry TIMESTAMPTZ NULL,
	availability_status user_availability_status DEFAULT 'offline' NOT NULL,
	last_active_at TIMESTAMPTZ NULL,
	last_login_at TIMESTAMPTZ NULL,
    CONSTRAINT constraint_users_on_country CHECK (LENGTH(country) <= 140),
    CONSTRAINT constraint_users_on_phone_number CHECK (LENGTH(phone_number) <= 20),
	CONSTRAINT constraint_users_on_phone_number_calling_code CHECK (LENGTH(phone_number_calling_code) <= 10),
    CONSTRAINT constraint_users_on_email_length CHECK (LENGTH(email) <= 320),
    CONSTRAINT constraint_users_on_first_name CHECK (LENGTH(first_name) <= 140),
    CONSTRAINT constraint_users_on_last_name CHECK (LENGTH(last_name) <= 140)
);
CREATE UNIQUE INDEX index_unique_users_on_email_and_type_when_deleted_at_is_null ON users (email, type)
WHERE deleted_at IS NULL;
CREATE INDEX index_tgrm_users_on_email ON users USING GIN (email gin_trgm_ops);

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

DROP TABLE IF EXISTS conversation_statuses CASCADE;
CREATE TABLE conversation_statuses (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL UNIQUE
);

DROP TABLE IF EXISTS conversation_priorities CASCADE;
CREATE TABLE conversation_priorities (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL UNIQUE
);

DROP TABLE IF EXISTS contact_channels CASCADE;
CREATE TABLE contact_channels (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),

	-- Cascade deletes when contact or inbox is deleted.
	contact_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	inbox_id INT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE,

	identifier TEXT NOT NULL,
	CONSTRAINT constraint_contact_channels_on_identifier CHECK (length(identifier) <= 1000),
	CONSTRAINT constraint_contact_channels_on_inbox_id_and_contact_id_unique UNIQUE (inbox_id, contact_id)
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
	sla_policy_id INT REFERENCES sla_policies(id) ON DELETE SET NULL ON UPDATE CASCADE,

    -- Cascade deletes when inbox is deleted.
	inbox_id INT REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,

	-- Restrict delete.
	contact_channel_id INT REFERENCES contact_channels(id) ON DELETE RESTRICT ON UPDATE CASCADE NOT NULL,
	status_id INT REFERENCES conversation_statuses(id) ON DELETE RESTRICT ON UPDATE CASCADE NOT NULL,
    priority_id INT REFERENCES conversation_priorities(id) ON DELETE RESTRICT ON UPDATE CASCADE,

	meta JSONB DEFAULT '{}'::jsonb NOT NULL,
	custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL,
    assignee_last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    first_reply_at TIMESTAMPTZ NULL,
    last_reply_at TIMESTAMPTZ NULL,
    closed_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,

	"subject" TEXT NULL,
	waiting_since TIMESTAMPTZ NULL,
	last_message_at TIMESTAMPTZ NULL,
	last_message TEXT NULL,
	last_message_sender message_sender_type NULL,
	next_sla_deadline_at TIMESTAMPTZ NULL,
	snoozed_until TIMESTAMPTZ NULL
);
CREATE INDEX index_conversations_on_assigned_user_id ON conversations (assigned_user_id);
CREATE INDEX index_conversations_on_assigned_team_id ON conversations (assigned_team_id);
CREATE INDEX index_conversations_on_snoozed_until ON conversations (snoozed_until);
CREATE INDEX index_conversations_on_contact_id ON conversations (contact_id);
CREATE INDEX index_conversations_on_inbox_id ON conversations (inbox_id);
CREATE INDEX index_conversations_on_status_id ON conversations (status_id);
CREATE INDEX index_conversations_on_priority_id ON conversations (priority_id);
CREATE INDEX index_conversations_on_created_at ON conversations (created_at);
CREATE INDEX index_conversations_on_last_message_at ON conversations (last_message_at);
CREATE INDEX index_conversations_on_next_sla_deadline_at ON conversations (next_sla_deadline_at);
CREATE INDEX index_conversations_on_waiting_since ON conversations (waiting_since);

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
CREATE INDEX index_trgm_conversation_messages_on_text_content ON conversation_messages USING GIN (text_content gin_trgm_ops);
CREATE INDEX index_conversation_messages_on_conversation_id ON conversation_messages (conversation_id);
CREATE INDEX index_conversation_messages_on_created_at ON conversation_messages (created_at);
CREATE INDEX index_conversation_messages_on_source_id ON conversation_messages (source_id);
CREATE INDEX index_conversation_messages_on_status ON conversation_messages (status);

DROP TABLE IF EXISTS automation_rules CASCADE;
CREATE TABLE automation_rules (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "name" TEXT NOT NULL,
    description TEXT NULL,
    "type" VARCHAR NOT NULL,
    rules JSONB NULL,
    events TEXT[] DEFAULT '{}'::TEXT[] NOT NULL,
    enabled BOOL DEFAULT TRUE NOT NULL,
	weight INT DEFAULT 0 NOT NULL,
	execution_mode automation_execution_mode DEFAULT 'all' NOT NULL,
    CONSTRAINT constraint_automation_rules_on_name CHECK (length("name") <= 140),
    CONSTRAINT constraint_automation_rules_on_description CHECK (length(description) <= 300)
);
CREATE INDEX index_automation_rules_on_enabled_and_weight ON automation_rules(enabled, weight);
CREATE INDEX index_automation_rules_on_type_and_weight ON automation_rules(type, weight);

DROP TABLE IF EXISTS macros CASCADE;
CREATE TABLE macros (
   id SERIAL PRIMARY KEY,
   created_at TIMESTAMPTZ DEFAULT NOW(),
   updated_at TIMESTAMPTZ DEFAULT NOW(),
   name TEXT NOT NULL,
   actions JSONB DEFAULT '{}'::jsonb NOT NULL,
   visibility macro_visibility NOT NULL,
   visible_when macro_visible_when[] NOT NULL DEFAULT ARRAY['replying', 'starting_conversation', 'adding_private_note']::macro_visible_when[],
   message_content TEXT NOT NULL,
   -- Cascade deletes when user is deleted.
   user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
   team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
   usage_count INT DEFAULT 0 NOT NULL,
   CONSTRAINT name_length CHECK (length(name) <= 140),
   CONSTRAINT message_content_length CHECK (length(message_content) <= 5000)
);

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
	CONSTRAINT constraint_media_on_filename CHECK (length(filename) <= 1000),
	CONSTRAINT constraint_media_on_content_id CHECK (length(content_id) <= 300)
);
CREATE INDEX index_media_on_model_type_and_model_id ON media(model_type, model_id);
CREATE INDEX index_media_on_content_id ON media(content_id);

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

DROP TABLE IF EXISTS tags CASCADE;
CREATE TABLE tags (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL UNIQUE,
	CONSTRAINT constraint_tags_on_name CHECK (length("name") <= 140)
);

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

DROP TABLE IF EXISTS conversation_tags CASCADE;
CREATE TABLE conversation_tags (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	-- Cascade deletes when tag or conversation is deleted.
	tag_id INT REFERENCES tags(id) ON DELETE CASCADE ON UPDATE CASCADE,
	conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE UNIQUE INDEX index_conversation_tags_on_conversation_id_and_tag_id ON conversation_tags (conversation_id, tag_id);

DROP TABLE IF EXISTS csat_responses CASCADE;
CREATE TABLE csat_responses (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
	uuid UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE,

	-- Cascade deletes when conversation is deleted.
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,

    rating INT DEFAULT 0 NOT NULL,
    feedback TEXT NULL,
    response_timestamp TIMESTAMPTZ NULL,
    CONSTRAINT constraint_csat_responses_on_rating CHECK (rating >= 0 AND rating <= 5),
    CONSTRAINT constraint_csat_responses_on_feedback CHECK (length(feedback) <= 1000)
);
CREATE INDEX index_csat_responses_on_uuid ON csat_responses(uuid);

DROP TABLE IF EXISTS views CASCADE;
CREATE TABLE views (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL,
    filters JSONB NOT NULL,
	-- Delete user views when user is deleted.
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT constraint_views_on_name CHECK (length(name) <= 140)
);
CREATE INDEX index_views_on_user_id ON views(user_id);

DROP TABLE IF EXISTS applied_slas CASCADE;
CREATE TABLE applied_slas (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),

	status applied_sla_status DEFAULT 'pending' NOT NULL,

	-- Cascade deletes when conversation or SLA policy is deleted.
	conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	sla_policy_id INT REFERENCES sla_policies(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,

	first_response_deadline_at TIMESTAMPTZ NULL,
	resolution_deadline_at TIMESTAMPTZ NULL,
	first_response_breached_at TIMESTAMPTZ NULL,
	resolution_breached_at TIMESTAMPTZ NULL,
	first_response_met_at TIMESTAMPTZ NULL,
	resolution_met_at TIMESTAMPTZ NULL
);
CREATE INDEX index_applied_slas_on_conversation_id ON applied_slas(conversation_id);
CREATE INDEX index_applied_slas_on_status ON applied_slas(status);

DROP TABLE IF EXISTS sla_events CASCADE;
CREATE TABLE sla_events (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	status sla_event_status DEFAULT 'pending' NOT NULL,
	applied_sla_id BIGINT REFERENCES applied_slas(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	sla_policy_id INT REFERENCES sla_policies(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	type sla_metric NOT NULL,
	deadline_at TIMESTAMPTZ NOT NULL,
	met_at TIMESTAMPTZ,
	breached_at TIMESTAMPTZ
);
CREATE INDEX index_sla_events_on_applied_sla_id ON sla_events(applied_sla_id);
CREATE INDEX index_sla_events_on_status ON sla_events(status);

DROP TABLE IF EXISTS scheduled_sla_notifications CASCADE;
CREATE TABLE scheduled_sla_notifications (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  applied_sla_id BIGINT NOT NULL REFERENCES applied_slas(id) ON DELETE CASCADE,
  sla_event_id BIGINT REFERENCES sla_events(id) ON DELETE CASCADE,
  metric sla_metric NOT NULL,
  notification_type sla_notification_type NOT NULL,
  recipients TEXT[] NOT NULL,
  send_at TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ
);
CREATE INDEX index_scheduled_sla_notifications_on_send_at ON scheduled_sla_notifications(send_at);
CREATE INDEX index_scheduled_sla_notifications_on_processed_at ON scheduled_sla_notifications(processed_at);

DROP TABLE IF EXISTS ai_providers CASCADE;
CREATE TABLE ai_providers (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	name TEXT NOT NULL UNIQUE,
	provider ai_provider NOT NULL,
	config JSONB NOT NULL DEFAULT '{}',
	is_default BOOLEAN NOT NULL DEFAULT FALSE,
	CONSTRAINT constraint_ai_providers_on_name CHECK (length(name) <= 140)
);
CREATE UNIQUE INDEX index_unique_ai_providers_on_is_default_when_is_default_is_true ON ai_providers USING btree (is_default)
WHERE (is_default = true);

DROP TABLE IF EXISTS ai_prompts CASCADE;
CREATE TABLE ai_prompts (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	title TEXT NOT NULL,
    key TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
	CONSTRAINT constraint_prompts_on_title CHECK (length(title) <= 140),
    CONSTRAINT constraint_prompts_on_key CHECK (length(key) <= 140)
);
CREATE INDEX index_ai_prompts_on_key ON ai_prompts USING btree (key);

DROP TABLE IF EXISTS custom_attribute_definitions CASCADE;
CREATE TABLE custom_attribute_definitions (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	description TEXT NOT NULL,
	applies_to TEXT NOT NULL,
	key TEXT NOT NULL,
	values TEXT[] DEFAULT '{}'::TEXT[] NOT NULL,
	data_type TEXT NOT NULL,
	regex TEXT NULL,
	regex_hint TEXT NULL,
	CONSTRAINT constraint_custom_attribute_definitions_on_name CHECK (length("name") <= 140),
	CONSTRAINT constraint_custom_attribute_definitions_on_description CHECK (length(description) <= 300),
	CONSTRAINT constraint_custom_attribute_definitions_on_key CHECK (length(key) <= 140),
	CONSTRAINT constraint_custom_attribute_definitions_on_applies_to CHECK (length(applies_to) <= 50),
	CONSTRAINT constraint_custom_attribute_definitions_on_data_type CHECK (length(data_type) <= 100),
	CONSTRAINT constraint_custom_attribute_definitions_on_regex CHECK (length(regex) <= 1000),
	CONSTRAINT constraint_custom_attribute_definitions_on_regex_hint CHECK (length(regex_hint) <= 1000),
	CONSTRAINT constraint_custom_attribute_definitions_key_applies_to_unique UNIQUE (key, applies_to)
);

DROP TABLE IF EXISTS contact_notes CASCADE;
CREATE TABLE contact_notes (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	contact_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	note TEXT NOT NULL,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX index_contact_notes_on_contact_id_created_at ON contact_notes (contact_id, created_at);

DROP TABLE IF EXISTS activity_logs CASCADE;
CREATE TABLE activity_logs (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	activity_type activity_log_type NOT NULL,
	activity_description TEXT NOT NULL,
	actor_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	target_model_type TEXT NOT NULL,
	target_model_id BIGINT NOT NULL,
	ip INET
);
CREATE INDEX IF NOT EXISTS index_activity_logs_on_actor_id ON activity_logs (actor_id);
CREATE INDEX IF NOT EXISTS index_activity_logs_on_activity_type ON activity_logs (activity_type);
CREATE INDEX IF NOT EXISTS index_activity_logs_on_created_at ON activity_logs (created_at);

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
CREATE INDEX index_webhooks_on_created_at ON webhooks (created_at);
CREATE INDEX index_webhooks_on_events ON webhooks USING GIN (events);

INSERT INTO ai_providers
("name", provider, config, is_default)
VALUES('openai', 'openai', '{"api_key": ""}'::jsonb, true);

-- Default AI prompts
INSERT INTO ai_prompts ("key", "content", title)
VALUES
('make_friendly', 'Modify the text to make it more friendly and approachable.', 'Make Friendly'),
('make_concise', 'Simplify the text to make it more concise and to the point.', 'Make Concise'),
('add_empathy', 'Add empathy to the text while retaining the original meaning.', 'Add Empathy'),
('adjust_positive_tone', 'Adjust the tone of the text to make it sound more positive and reassuring.', 'Adjust Positive Tone'),
('make_professional', 'Rephrase the text to make it sound more formal and professional and to the point.', 'Make Professional');

-- Default settings
INSERT INTO settings ("key", value)
VALUES
    ('app.lang', '"en"'::jsonb),
    ('app.root_url', '"http://localhost:9000"'::jsonb),
    ('app.logo_url', '"http://localhost:9000/logo.png"'::jsonb),
    ('app.site_name', '"Libredesk"'::jsonb),
    ('app.favicon_url', '"http://localhost:9000/favicon.ico"'::jsonb),
    ('app.max_file_upload_size', '20'::jsonb),
    ('app.allowed_file_upload_extensions', '["*"]'::jsonb),
	('app.timezone', '"Asia/Kolkata"'::jsonb),
	('app.business_hours_id', '""'::jsonb),
    ('notification.email.username', '"admin@yourcompany.com"'::jsonb),
    ('notification.email.host', '"smtp.gmail.com"'::jsonb),
    ('notification.email.port', '587'::jsonb),
    ('notification.email.password', '""'::jsonb),
    ('notification.email.max_conns', '5'::jsonb),
    ('notification.email.idle_timeout', '"25s"'::jsonb),
    ('notification.email.wait_timeout', '"60s"'::jsonb),
    ('notification.email.auth_protocol', '"plain"'::jsonb),
	('notification.email.tls_type', '"starttls"'::jsonb),
	('notification.email.tls_skip_verify', 'false'::jsonb),
	('notification.email.hello_hostname', '""'::jsonb),
    ('notification.email.email_address', '"admin@yourcompany.com"'::jsonb),
    ('notification.email.max_msg_retries', '3'::jsonb),
    ('notification.email.enabled', 'false'::jsonb);

-- Default conversation priorities
INSERT INTO conversation_priorities (name) VALUES
('Low'),
('Medium'),
('High');

-- Default conversation statuses
INSERT INTO conversation_statuses (name) VALUES
('Open'),
('Snoozed'),
('Resolved'),
('Closed');

-- Default roles
INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Agent',
		'Role for all agents with limited access to conversations.',
		'{conversations:read_all,conversations:read_unassigned,conversations:read_assigned,conversations:read_team_inbox,conversations:read,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write,view:manage}'
	);

INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Admin',
		'Role for users who have complete access to everything.',
		'{activity_logs:manage,custom_attributes:manage,contacts:read_all,contacts:read,contacts:write,contacts:block,contact_notes:read,contact_notes:write,contact_notes:delete,conversations:write,ai:manage,general_settings:manage,notification_settings:manage,oidc:manage,conversations:read_all,conversations:read_unassigned,conversations:read_assigned,conversations:read_team_inbox,conversations:read,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write,view:manage,status:manage,tags:manage,macros:manage,users:manage,teams:manage,automations:manage,inboxes:manage,roles:manage,reports:manage,templates:manage,business_hours:manage,sla:manage}'
	);


-- Email notification templates
INSERT INTO templates
("type", body, is_default, "name", subject, is_builtin)
VALUES('email_notification'::template_type, '
<p>A new conversation has been assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'Conversation assigned', 'New conversation assigned to you', true);

INSERT INTO templates
("type", body, is_default, "name", subject, is_builtin)
VALUES (
  'email_notification'::template_type,
  '

<p>This is a notification that the SLA for conversation {{ .Conversation.ReferenceNumber }} is approaching the SLA deadline for {{ .SLA.Metric }}.</p>

<p>
  Details:<br>
  - Conversation reference number: {{ .Conversation.ReferenceNumber }}<br>
  - Metric: {{ .SLA.Metric }}<br>
  - Due in: {{ .SLA.DueIn }}
</p>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>


<p>
  Best regards,<br>
  Libredesk
</p>

',
  false,
  'SLA breach warning',
  'SLA Alert: Conversation {{ .Conversation.ReferenceNumber }} is approaching SLA deadline for {{ .SLA.Metric }}',
  true
);

INSERT INTO templates
("type", body, is_default, "name", subject, is_builtin)
VALUES (
  'email_notification'::template_type,
  '
<p>This is an urgent alert that the SLA for conversation {{ .Conversation.ReferenceNumber }} has been breached for {{ .SLA.Metric }}. Please take immediate action.</p>

<p>
  Details:<br>
  - Conversation reference number: {{ .Conversation.ReferenceNumber }}<br>
  - Metric: {{ .SLA.Metric }}<br>
  - Overdue by: {{ .SLA.OverdueBy }}
</p>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>


<p>
  Best regards,<br>
  Libredesk
</p>

',
  false,
  'SLA breached',
  'Urgent: SLA Breach for Conversation {{ .Conversation.ReferenceNumber }} for {{ .SLA.Metric }}',
  true
);
