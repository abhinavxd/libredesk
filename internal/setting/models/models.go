package models

type General struct {
	SiteName                    string   `json:"app.site_name"`
	Lang                        string   `json:"app.lang"`
	MaxFileUploadSize           int      `json:"app.max_file_upload_size"`
	FaviconURL                  string   `json:"app.favicon_url"`
	LogoURL                     string   `json:"app.logo_url"`
	RootURL                     string   `json:"app.root_url"`
	AllowedFileUploadExtensions []string `json:"app.allowed_file_upload_extensions"`
	Timezone                    string   `json:"app.timezone"`
	BusinessHoursID             string   `json:"app.business_hours_id"`
}

type EmailNotification struct {
	Username      string `json:"notification.email.username" db:"notification.email.username"`
	Host          string `json:"notification.email.host" db:"notification.email.host"`
	Port          int    `json:"notification.email.port" db:"notification.email.port"`
	Password      string `json:"notification.email.password" db:"notification.email.password"`
	MaxConns      int    `json:"notification.email.max_conns" db:"notification.email.max_conns"`
	IdleTimeout   string `json:"notification.email.idle_timeout" db:"notification.email.idle_timeout"`
	WaitTimeout   string `json:"notification.email.wait_timeout" db:"notification.email.wait_timeout"`
	AuthProtocol  string `json:"notification.email.auth_protocol" db:"notification.email.auth_protocol"`
	EmailAddress  string `json:"notification.email.email_address" db:"notification.email.email_address"`
	MaxMsgRetries int    `json:"notification.email.max_msg_retries" db:"notification.email.max_msg_retries"`
	TLSType       string `json:"notification.email.tls_type" db:"notification.email.tls_type"`
	TLSSkipVerify bool   `json:"notification.email.tls_skip_verify" db:"notification.email.tls_skip_verify"`
	HelloHostname string `json:"notification.email.hello_hostname" db:"notification.email.hello_hostname"`
	Enabled       bool   `json:"notification.email.enabled" db:"notification.email.enabled"`
}

type Settings struct {
	EmailNotification
	General
}

// AISettings holds AI/RAG configuration.
type AISettings struct {
	Enabled             bool    `json:"ai.enabled" db:"ai.enabled"`
	Provider            string  `json:"ai.provider" db:"ai.provider"`
	OpenAIAPIKey        string  `json:"ai.openai_api_key" db:"ai.openai_api_key"`
	OpenRouterAPIKey    string  `json:"ai.openrouter_api_key" db:"ai.openrouter_api_key"`
	OpenRouterModel     string  `json:"ai.openrouter_model" db:"ai.openrouter_model"`
	EmbeddingModel      string  `json:"ai.embedding_model" db:"ai.embedding_model"`
	SystemPrompt        string  `json:"ai.system_prompt" db:"ai.system_prompt"`
	MaxContextChunks    int     `json:"ai.max_context_chunks" db:"ai.max_context_chunks"`
	SimilarityThreshold float64 `json:"ai.similarity_threshold" db:"ai.similarity_threshold"`
}
