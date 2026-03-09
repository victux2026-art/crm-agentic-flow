package domain

import (
	"time"
)

// Person representa a un individuo en el CRM
type Person struct {
	ID              int                    `json:"id"`
	FirstName       string                 `json:"first_name"`
	LastName        string                 `json:"last_name"`
	Email           string                 `json:"email"`
	PhoneNumber     *string                `json:"phone_number"`     // Nulable
	CompanyName     *string                `json:"company_name"`     // Nulable
	JobTitle        *string                `json:"job_title"`        // Nulable
	LinkedinProfile *string                `json:"linkedin_profile"` // Nulable
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
	Embedding       []float32              `json:"embedding"`
}

// Flow representa un proceso de venta o flujo de trabajo
type Flow struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	PersonID          int                    `json:"person_id"`
	CompanyName       *string                `json:"company_name"` // Nulable
	Status            string                 `json:"status"`
	Value             float64                `json:"value"`
	Currency          string                 `json:"currency"`
	ExpectedCloseDate *time.Time             `json:"expected_close_date"` // Nulable
	ActualCloseDate   *time.Time             `json:"actual_close_date"`   // Nulable
	Priority          string                 `json:"priority"`
	HealthScore       int                    `json:"health_score"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	Metadata          map[string]interface{} `json:"metadata"`
	Embedding         []float32              `json:"embedding"`
}

// Activity representa una tarea o interacción ligada a un flujo
type Activity struct {
	ID          int                    `json:"id"`
	FlowID      *int                   `json:"flow_id"`   // Nulable
	PersonID    *int                   `json:"person_id"` // Nulable
	Type        string                 `json:"type"`
	Subject     *string                `json:"subject"`      // Nulable
	Description *string                `json:"description"`  // Nulable
	DueDate     *time.Time             `json:"due_date"`     // Nulable
	CompletedAt *time.Time             `json:"completed_at"` // Nulable
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
	Embedding   []float32              `json:"embedding"`
}

// Event representa un evento asíncrono para el sistema de agentes
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   []byte    `json:"payload"`
	Retries   int       `json:"retries"`
}
