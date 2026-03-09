package main

import "time"

type Organization struct {
	ID          int64                  `json:"id"`
	TenantID    int64                  `json:"tenant_id"`
	Name        string                 `json:"name"`
	Domain      *string                `json:"domain"`
	Industry    *string                `json:"industry"`
	SizeBand    *string                `json:"size_band"`
	Country     *string                `json:"country"`
	OwnerUserID *int64                 `json:"owner_user_id"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type MVPPerson struct {
	ID             int64                  `json:"id"`
	TenantID       int64                  `json:"tenant_id"`
	OrganizationID *int64                 `json:"organization_id"`
	FirstName      string                 `json:"first_name"`
	LastName       string                 `json:"last_name"`
	Email          *string                `json:"email"`
	Phone          *string                `json:"phone"`
	JobTitle       *string                `json:"job_title"`
	LinkedinURL    *string                `json:"linkedin_url"`
	Status         string                 `json:"status"`
	OwnerUserID    *int64                 `json:"owner_user_id"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type Deal struct {
	ID                int64                  `json:"id"`
	TenantID          int64                  `json:"tenant_id"`
	OrganizationID    *int64                 `json:"organization_id"`
	PrimaryPersonID   *int64                 `json:"primary_person_id"`
	Name              string                 `json:"name"`
	Stage             string                 `json:"stage"`
	Status            string                 `json:"status"`
	ValueAmount       float64                `json:"value_amount"`
	ValueCurrency     string                 `json:"value_currency"`
	CloseDateExpected *time.Time             `json:"close_date_expected"`
	CloseDateActual   *time.Time             `json:"close_date_actual"`
	OwnerUserID       *int64                 `json:"owner_user_id"`
	HealthScore       *int                   `json:"health_score"`
	Source            *string                `json:"source"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type Task struct {
	ID              int64      `json:"id"`
	TenantID        int64      `json:"tenant_id"`
	DealID          *int64     `json:"deal_id"`
	PersonID        *int64     `json:"person_id"`
	OrganizationID  *int64     `json:"organization_id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description"`
	Status          string     `json:"status"`
	Priority        string     `json:"priority"`
	DueAt           *time.Time `json:"due_at"`
	OwnerUserID     *int64     `json:"owner_user_id"`
	CreatedByUserID *int64     `json:"created_by_user_id"`
	Source          string     `json:"source"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Note struct {
	ID             int64      `json:"id"`
	TenantID       int64      `json:"tenant_id"`
	DealID         *int64     `json:"deal_id"`
	PersonID       *int64     `json:"person_id"`
	OrganizationID *int64     `json:"organization_id"`
	AuthorUserID   *int64     `json:"author_user_id"`
	Body           string     `json:"body"`
	Source         string     `json:"source"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type WebhookEndpoint struct {
	ID              int64     `json:"id"`
	TenantID        int64     `json:"tenant_id"`
	Name            string    `json:"name"`
	TargetURL       string    `json:"target_url"`
	SigningSecret   string    `json:"signing_secret"`
	Status          string    `json:"status"`
	CreatedByUserID *int64    `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type WebhookSubscription struct {
	ID                int64     `json:"id"`
	TenantID          int64     `json:"tenant_id"`
	WebhookEndpointID int64     `json:"webhook_endpoint_id"`
	EventType         string    `json:"event_type"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
}

type WebhookDelivery struct {
	ID                int64      `json:"id"`
	TenantID          int64      `json:"tenant_id"`
	WebhookEndpointID int64      `json:"webhook_endpoint_id"`
	OutboxEventID     *int64     `json:"outbox_event_id"`
	Status            string     `json:"status"`
	AttemptCount      int        `json:"attempt_count"`
	HTTPStatus        *int       `json:"http_status"`
	LastError         *string    `json:"last_error"`
	ResponseBody      *string    `json:"response_body"`
	DeliveredAt       *time.Time `json:"delivered_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

type WebhookDeliveryStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type WebhookDeliveryEndpointCount struct {
	WebhookEndpointID int64  `json:"webhook_endpoint_id"`
	EndpointName      string `json:"endpoint_name"`
	Count             int    `json:"count"`
}

type WebhookDeliveryStats struct {
	ByStatus       []WebhookDeliveryStatusCount   `json:"by_status"`
	ByEndpoint     []WebhookDeliveryEndpointCount `json:"by_endpoint"`
	RecentFailed   []WebhookDelivery              `json:"recent_failed"`
	RecentDelivered []WebhookDelivery             `json:"recent_delivered"`
}

type OutboxEvent struct {
	ID            int64                  `json:"id"`
	TenantID      *int64                 `json:"tenant_id"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload"`
	Status        string                 `json:"status"`
	AttemptCount  int                    `json:"attempt_count"`
	NextAttemptAt time.Time              `json:"next_attempt_at"`
	LastError     *string                `json:"last_error"`
	CreatedAt     time.Time              `json:"created_at"`
	ProcessedAt   *time.Time             `json:"processed_at"`
}

type OutboxStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type OutboxEventTypeCount struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

type OutboxStats struct {
	ByStatus     []OutboxStatusCount    `json:"by_status"`
	ByEventType  []OutboxEventTypeCount `json:"by_event_type"`
	RecentFailed []OutboxEvent          `json:"recent_failed"`
	NextRetry    []OutboxEvent          `json:"next_retry"`
}

type AuditEntry struct {
	ID         int64                  `json:"id"`
	TenantID   *int64                 `json:"tenant_id"`
	ActorType  *string                `json:"actor_type"`
	ActorID    *int64                 `json:"actor_id"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   int64                  `json:"entity_id"`
	Changes    map[string]interface{} `json:"changes"`
	IPAddress  *string                `json:"ip_address"`
	UserAgent  *string                `json:"user_agent"`
	CreatedAt  time.Time              `json:"created_at"`
}
