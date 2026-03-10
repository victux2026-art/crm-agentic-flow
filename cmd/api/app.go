package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"os"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func defaultDatabaseURL() string {
	return "postgresql://postgres:postgres@localhost:5440/crm_agentic_flow"
}

func initDBPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	if dbURL == "" {
		dbURL = defaultDatabaseURL()
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func initJWTSecret() {
	if os.Getenv("JWT_SECRET") == "" {
		randBytes := make([]byte, 32)
		if _, err := rand.Read(randBytes); err != nil {
			log.Fatalf("Error generating JWT secret: %v\n", err)
		}
		jwtSecret = randBytes
		return
	}

	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
}

// compatibilityDeprecationMiddleware marks legacy routes that only exist during
// the migration from persons/flows/activities to people/deals/tasks.
func compatibilityDeprecationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", "Wed, 30 Sep 2026 23:59:59 GMT")
		w.Header().Set("Link", `</people>; rel="successor-version"`)
		w.Header().Set("X-API-Compatibility", "legacy")
		next.ServeHTTP(w, r)
	})
}

func legacyAPIEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("ENABLE_LEGACY_API")))
	switch value {
	case "", "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func newRouter() http.Handler {
	return newRouterWithOptions(legacyAPIEnabled())
}

func newRouterWithOptions(includeLegacy bool) http.Handler {
	r := chi.NewRouter()

	mountUIRoutes(r)
	r.Get("/health", healthHandler)
	r.Post("/login", loginHandler)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		if includeLegacy {
			r.Group(func(r chi.Router) {
				r.Use(compatibilityDeprecationMiddleware)

				r.Post("/persons", createPerson)
				r.Get("/persons", getPersons)
				r.Get("/persons/{id}", getPersonByID)
				r.Put("/persons/{id}", updatePerson)
				r.Delete("/persons/{id}", deletePerson)

				r.Post("/flows", createFlow)
				r.Get("/flows", getFlows)
				r.Get("/flows/{id}", getFlowByID)
				r.Put("/flows/{id}", updateFlow)
				r.Delete("/flows/{id}", deleteFlow)

				r.Post("/activities", createActivity)
				r.Get("/activities", getActivities)
				r.Get("/activities/{id}", getActivityByID)
				r.Put("/activities/{id}", updateActivity)
				r.Delete("/activities/{id}", deleteActivity)
			})
		}

		r.Post("/organizations", createOrganization)
		r.Get("/organizations", getOrganizations)
		r.Get("/organizations/{id}", getOrganizationByID)
		r.Put("/organizations/{id}", updateOrganization)
		r.Delete("/organizations/{id}", deleteOrganization)
		r.Post("/people", createMVPPerson)
		r.Get("/people", getMVPPeople)
		r.Get("/people/{id}", getMVPPersonByID)
		r.Put("/people/{id}", updateMVPPerson)
		r.Delete("/people/{id}", deleteMVPPerson)
		r.Post("/deals", createDeal)
		r.Get("/deals", getDeals)
		r.Get("/deals/{id}", getDealByID)
		r.Put("/deals/{id}", updateDeal)
		r.Delete("/deals/{id}", deleteDeal)
		r.Post("/tasks", createTask)
		r.Get("/tasks", getTasks)
		r.Get("/tasks/{id}", getTaskByID)
		r.Put("/tasks/{id}", updateTask)
		r.Delete("/tasks/{id}", deleteTask)
		r.Post("/notes", createNote)
		r.Get("/notes", getNotes)
		r.Get("/notes/{id}", getNoteByID)
		r.Put("/notes/{id}", updateNote)
		r.Delete("/notes/{id}", deleteNote)
		r.Get("/admin/tenant", getAdminTenant)
		r.Put("/admin/tenant", updateAdminTenant)
		r.Get("/admin/users", getAdminUsers)
		r.Post("/admin/users", createAdminUser)
		r.Put("/admin/users/{id}", updateAdminUser)
		r.Post("/webhook-endpoints", createWebhookEndpoint)
		r.Get("/webhook-endpoints", getWebhookEndpoints)
		r.Put("/webhook-endpoints/{id}", updateWebhookEndpoint)
		r.Delete("/webhook-endpoints/{id}", deleteWebhookEndpoint)
		r.Post("/webhook-subscriptions", createWebhookSubscription)
		r.Get("/webhook-subscriptions", getWebhookSubscriptions)
		r.Put("/webhook-subscriptions/{id}", updateWebhookSubscription)
		r.Delete("/webhook-subscriptions/{id}", deleteWebhookSubscription)
		r.Get("/webhook-deliveries", getWebhookDeliveries)
		r.Get("/webhook-deliveries/stats", getWebhookDeliveryStats)
		r.Get("/outbox-events", getOutboxEvents)
		r.Get("/outbox-events/stats", getOutboxStats)
		r.Get("/outbox-events/{id}", getOutboxEventByID)
		r.Post("/webhook-deliveries/{id}/replay", replayWebhookDelivery)
		r.Post("/outbox-events/{id}/replay", replayOutboxEvent)
		r.Post("/outbox-events/replay", replayOutboxEventsByFilter)
		r.Get("/audit-log", getAuditLog)
	})

	return r
}
