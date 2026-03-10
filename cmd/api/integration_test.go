package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testPasswordHash = "$2a$10$Iag1qEZ08E5uPWumhniku.QxLzn3Far1wFXKLFY/GrM2o3nRKPWFa"

func TestLoginAndCreateOrganizationWritesOutboxAndAudit(t *testing.T) {
	ctx := context.Background()

	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = defaultDatabaseURL()
	}

	pool, err := initDBPool(ctx, testDBURL)
	if err != nil {
		t.Skipf("skipping integration test; database unavailable: %v", err)
	}
	defer pool.Close()

	prevPool := dbPool
	prevSecret := jwtSecret
	dbPool = pool
	jwtSecret = []byte("test-secret")
	defer func() {
		dbPool = prevPool
		jwtSecret = prevSecret
	}()

	resetIntegrationState(t, ctx, pool)
	seedIntegrationUser(t, ctx, pool)

	router := newRouter()

	token := loginTestUser(t, router)
	org := createTestOrganization(t, router, token)
	if org["name"] != "Integration Org" {
		t.Fatalf("unexpected organization name: %#v", org["name"])
	}

	var outboxCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE tenant_id = 1 AND event_type = 'organization.created'`).Scan(&outboxCount)
	if err != nil {
		t.Fatalf("query outbox_events: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 outbox event, got %d", outboxCount)
	}

	var auditCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM audit_log
		WHERE tenant_id = 1 AND action = 'created' AND entity_type = 'organization'`).Scan(&auditCount)
	if err != nil {
		t.Fatalf("query audit_log: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", auditCount)
	}
}

func TestUpdateOrganizationWritesOutboxAndAudit(t *testing.T) {
	ctx := context.Background()

	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = defaultDatabaseURL()
	}

	pool, err := initDBPool(ctx, testDBURL)
	if err != nil {
		t.Skipf("skipping integration test; database unavailable: %v", err)
	}
	defer pool.Close()

	prevPool := dbPool
	prevSecret := jwtSecret
	dbPool = pool
	jwtSecret = []byte("test-secret")
	defer func() {
		dbPool = prevPool
		jwtSecret = prevSecret
	}()

	resetIntegrationState(t, ctx, pool)
	seedIntegrationUser(t, ctx, pool)

	router := newRouter()
	token := loginTestUser(t, router)
	org := createTestOrganization(t, router, token)

	orgID := int64(org["id"].(float64))
	updateBody := map[string]interface{}{
		"name":     "Integration Org Updated",
		"domain":   "updated.integration.test",
		"industry": "QA",
		"metadata": map[string]interface{}{"source": "test"},
	}
	resp := performJSONRequest(t, router, http.MethodPut, "/organizations/"+itoa(orgID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update organization returned status %d: %s", resp.Code, resp.Body.String())
	}

	var outboxCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE tenant_id = 1 AND event_type = 'organization.updated'`).Scan(&outboxCount)
	if err != nil {
		t.Fatalf("query updated outbox_events: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 organization.updated outbox event, got %d", outboxCount)
	}

	var auditCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM audit_log
		WHERE tenant_id = 1 AND action = 'updated' AND entity_type = 'organization'`).Scan(&auditCount)
	if err != nil {
		t.Fatalf("query updated audit_log: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected 1 organization update audit log entry, got %d", auditCount)
	}
}

func TestCreateAndUpdatePersonWritesOutboxAndAudit(t *testing.T) {
	ctx := context.Background()

	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = defaultDatabaseURL()
	}

	pool, err := initDBPool(ctx, testDBURL)
	if err != nil {
		t.Skipf("skipping integration test; database unavailable: %v", err)
	}
	defer pool.Close()

	prevPool := dbPool
	prevSecret := jwtSecret
	dbPool = pool
	jwtSecret = []byte("test-secret")
	defer func() {
		dbPool = prevPool
		jwtSecret = prevSecret
	}()

	resetIntegrationState(t, ctx, pool)
	seedIntegrationUser(t, ctx, pool)
	router := newRouter()
	token := loginTestUser(t, router)
	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	personBody := map[string]interface{}{
		"organization_id": orgID,
		"first_name":      "Jane",
		"last_name":       "Doe",
		"email":           "jane@example.com",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/people", personBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create person returned status %d: %s", resp.Code, resp.Body.String())
	}

	var person map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &person); err != nil {
		t.Fatalf("decode person response: %v", err)
	}
	personID := int64(person["id"].(float64))

	updateBody := map[string]interface{}{
		"organization_id": orgID,
		"first_name":      "Jane",
		"last_name":       "Roe",
		"email":           "jane.roe@example.com",
		"status":          "active",
		"metadata":        map[string]interface{}{"segment": "beta"},
	}
	resp = performJSONRequest(t, router, http.MethodPut, "/people/"+itoa(personID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update person returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertOutboxCount(t, ctx, pool, "person.created", 1)
	assertOutboxCount(t, ctx, pool, "person.updated", 1)
	assertAuditCount(t, ctx, pool, "created", "person", 1)
	assertAuditCount(t, ctx, pool, "updated", "person", 1)
}

func TestCreateAndUpdateDealWritesOutboxAndAudit(t *testing.T) {
	ctx := context.Background()

	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = defaultDatabaseURL()
	}

	pool, err := initDBPool(ctx, testDBURL)
	if err != nil {
		t.Skipf("skipping integration test; database unavailable: %v", err)
	}
	defer pool.Close()

	prevPool := dbPool
	prevSecret := jwtSecret
	dbPool = pool
	jwtSecret = []byte("test-secret")
	defer func() {
		dbPool = prevPool
		jwtSecret = prevSecret
	}()

	resetIntegrationState(t, ctx, pool)
	seedIntegrationUser(t, ctx, pool)
	router := newRouter()
	token := loginTestUser(t, router)
	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	dealBody := map[string]interface{}{
		"organization_id": orgID,
		"name":            "Expansion Deal",
		"stage":           "proposal",
		"value_amount":    5000,
		"value_currency":  "USD",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/deals", dealBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create deal returned status %d: %s", resp.Code, resp.Body.String())
	}

	var deal map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &deal); err != nil {
		t.Fatalf("decode deal response: %v", err)
	}
	dealID := int64(deal["id"].(float64))

	updateBody := map[string]interface{}{
		"organization_id": orgID,
		"name":            "Expansion Deal Updated",
		"stage":           "negotiation",
		"status":          "open",
		"value_amount":    7500,
		"value_currency":  "USD",
		"metadata":        map[string]interface{}{"priority": "high"},
	}
	resp = performJSONRequest(t, router, http.MethodPut, "/deals/"+itoa(dealID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update deal returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertOutboxCount(t, ctx, pool, "deal.created", 1)
	assertOutboxCount(t, ctx, pool, "deal.updated", 1)
	assertAuditCount(t, ctx, pool, "created", "deal", 1)
	assertAuditCount(t, ctx, pool, "updated", "deal", 1)
}

func TestDeleteOrganizationWritesAudit(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	resp := performJSONRequest(t, router, http.MethodDelete, "/organizations/"+itoa(orgID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete organization returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertAuditCount(t, ctx, pool, "deleted", "organization", 1)
}

func TestDeletePersonWritesAudit(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	personBody := map[string]interface{}{
		"organization_id": orgID,
		"first_name":      "Delete",
		"last_name":       "Me",
		"email":           "deleteme@example.com",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/people", personBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create person returned status %d: %s", resp.Code, resp.Body.String())
	}

	var person map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &person); err != nil {
		t.Fatalf("decode person response: %v", err)
	}
	personID := int64(person["id"].(float64))

	resp = performJSONRequest(t, router, http.MethodDelete, "/people/"+itoa(personID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete person returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertAuditCount(t, ctx, pool, "deleted", "person", 1)
}

func TestDeleteDealWritesAudit(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	dealBody := map[string]interface{}{
		"organization_id": orgID,
		"name":            "Delete Deal",
		"stage":           "proposal",
		"value_amount":    1000,
		"value_currency":  "USD",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/deals", dealBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create deal returned status %d: %s", resp.Code, resp.Body.String())
	}

	var deal map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &deal); err != nil {
		t.Fatalf("decode deal response: %v", err)
	}
	dealID := int64(deal["id"].(float64))

	resp = performJSONRequest(t, router, http.MethodDelete, "/deals/"+itoa(dealID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete deal returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertAuditCount(t, ctx, pool, "deleted", "deal", 1)
}

func TestCreateUpdateAndDeleteTaskWritesOutboxAndAudit(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	taskBody := map[string]interface{}{
		"organization_id": orgID,
		"title":           "Call customer",
		"status":          "open",
		"priority":        "high",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/tasks", taskBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create task returned status %d: %s", resp.Code, resp.Body.String())
	}

	var task map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	taskID := int64(task["id"].(float64))

	updateBody := map[string]interface{}{
		"organization_id": orgID,
		"title":           "Call customer again",
		"status":          "completed",
		"priority":        "normal",
		"source":          "manual",
	}
	resp = performJSONRequest(t, router, http.MethodPut, "/tasks/"+itoa(taskID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update task returned status %d: %s", resp.Code, resp.Body.String())
	}

	resp = performJSONRequest(t, router, http.MethodDelete, "/tasks/"+itoa(taskID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete task returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertOutboxCount(t, ctx, pool, "task.created", 1)
	assertOutboxCount(t, ctx, pool, "task.updated", 1)
	assertAuditCount(t, ctx, pool, "created", "task", 1)
	assertAuditCount(t, ctx, pool, "updated", "task", 1)
	assertAuditCount(t, ctx, pool, "deleted", "task", 1)
}

func TestCreateUpdateAndDeleteNoteWritesOutboxAndAudit(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	org := createTestOrganization(t, router, token)
	orgID := int64(org["id"].(float64))

	noteBody := map[string]interface{}{
		"organization_id": orgID,
		"body":            "Initial note",
		"source":          "manual",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/notes", noteBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create note returned status %d: %s", resp.Code, resp.Body.String())
	}

	var note map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &note); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	noteID := int64(note["id"].(float64))

	updateBody := map[string]interface{}{
		"organization_id": orgID,
		"body":            "Updated note",
		"source":          "manual",
	}
	resp = performJSONRequest(t, router, http.MethodPut, "/notes/"+itoa(noteID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update note returned status %d: %s", resp.Code, resp.Body.String())
	}

	resp = performJSONRequest(t, router, http.MethodDelete, "/notes/"+itoa(noteID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete note returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertOutboxCount(t, ctx, pool, "note.created", 1)
	assertOutboxCount(t, ctx, pool, "note.updated", 1)
	assertAuditCount(t, ctx, pool, "created", "note", 1)
	assertAuditCount(t, ctx, pool, "updated", "note", 1)
	assertAuditCount(t, ctx, pool, "deleted", "note", 1)
}

func TestCreateAndListWebhookEndpoints(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	if endpoint["name"] != "Primary Endpoint" {
		t.Fatalf("unexpected webhook endpoint name: %#v", endpoint["name"])
	}

	resp := performJSONRequest(t, router, http.MethodGet, "/webhook-endpoints", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("list webhook endpoints returned status %d: %s", resp.Code, resp.Body.String())
	}

	var endpoints []map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &endpoints); err != nil {
		t.Fatalf("decode webhook endpoints response: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 webhook endpoint, got %d", len(endpoints))
	}
}

func TestUpdateAndDeleteWebhookEndpoint(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	endpointID := int64(endpoint["id"].(float64))

	updateBody := map[string]interface{}{
		"name":           "Primary Endpoint Updated",
		"target_url":     "https://example.test/new-webhook",
		"signing_secret": "super-secret-2",
		"status":         "inactive",
	}
	resp := performJSONRequest(t, router, http.MethodPut, "/webhook-endpoints/"+itoa(endpointID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update webhook endpoint returned status %d: %s", resp.Code, resp.Body.String())
	}

	var updated map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated webhook endpoint response: %v", err)
	}
	if updated["status"] != "inactive" {
		t.Fatalf("unexpected updated webhook endpoint status: %#v", updated["status"])
	}

	resp = performJSONRequest(t, router, http.MethodDelete, "/webhook-endpoints/"+itoa(endpointID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete webhook endpoint returned status %d: %s", resp.Code, resp.Body.String())
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_endpoints WHERE id = $1`, endpointID).Scan(&count); err != nil {
		t.Fatalf("query webhook endpoint count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected webhook endpoint to be deleted, got count %d", count)
	}
}

func TestCreateAndListWebhookSubscriptions(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	endpointID := int64(endpoint["id"].(float64))

	body := map[string]interface{}{
		"webhook_endpoint_id": endpointID,
		"event_type":          "organization.created",
	}
	resp := performJSONRequest(t, router, http.MethodPost, "/webhook-subscriptions", body, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create webhook subscription returned status %d: %s", resp.Code, resp.Body.String())
	}

	resp = performJSONRequest(t, router, http.MethodGet, "/webhook-subscriptions", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("list webhook subscriptions returned status %d: %s", resp.Code, resp.Body.String())
	}

	var subscriptions []map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &subscriptions); err != nil {
		t.Fatalf("decode webhook subscriptions response: %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("expected 1 webhook subscription, got %d", len(subscriptions))
	}
	if subscriptions[0]["event_type"] != "organization.created" {
		t.Fatalf("unexpected webhook subscription event_type: %#v", subscriptions[0]["event_type"])
	}
}

func TestUpdateAndDeleteWebhookSubscription(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	endpointID := int64(endpoint["id"].(float64))
	subscriptionID := createTestWebhookSubscription(t, router, token, endpointID, "organization.created")

	updateBody := map[string]interface{}{
		"webhook_endpoint_id": endpointID,
		"event_type":          "organization.updated",
		"is_active":           false,
	}
	resp := performJSONRequest(t, router, http.MethodPut, "/webhook-subscriptions/"+itoa(subscriptionID), updateBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update webhook subscription returned status %d: %s", resp.Code, resp.Body.String())
	}

	var updated map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated webhook subscription response: %v", err)
	}
	if updated["event_type"] != "organization.updated" {
		t.Fatalf("unexpected updated webhook subscription event_type: %#v", updated["event_type"])
	}
	if updated["is_active"] != false {
		t.Fatalf("unexpected updated webhook subscription is_active: %#v", updated["is_active"])
	}

	resp = performJSONRequest(t, router, http.MethodDelete, "/webhook-subscriptions/"+itoa(subscriptionID), nil, token)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete webhook subscription returned status %d: %s", resp.Code, resp.Body.String())
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_subscriptions WHERE id = $1`, subscriptionID).Scan(&count); err != nil {
		t.Fatalf("query webhook subscription count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected webhook subscription to be deleted, got count %d", count)
	}
}

func TestGetWebhookDeliveriesSupportsFilters(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	endpointID := int64(endpoint["id"].(float64))
	outboxEventID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "processed")
	insertTestWebhookDelivery(t, ctx, pool, 1, endpointID, outboxEventID, "failed", 2)

	resp := performJSONRequest(t, router, http.MethodGet, "/webhook-deliveries?status=failed&event_type=organization.created&webhook_endpoint_id="+itoa(endpointID), nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("list filtered webhook deliveries returned status %d: %s", resp.Code, resp.Body.String())
	}

	var deliveries []map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &deliveries); err != nil {
		t.Fatalf("decode webhook deliveries response: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 webhook delivery, got %d", len(deliveries))
	}
	if deliveries[0]["status"] != "failed" {
		t.Fatalf("unexpected webhook delivery status: %#v", deliveries[0]["status"])
	}
}

func TestReplayWebhookDeliveryResetsDeliveryAndOutbox(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	endpointID := int64(endpoint["id"].(float64))
	outboxEventID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "failed")
	deliveryID := insertTestWebhookDelivery(t, ctx, pool, 1, endpointID, outboxEventID, "failed", 3)

	resp := performJSONRequest(t, router, http.MethodPost, "/webhook-deliveries/"+itoa(deliveryID)+"/replay", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("replay webhook delivery returned status %d: %s", resp.Code, resp.Body.String())
	}

	var deliveryStatus string
	var httpStatus *int
	var lastError *string
	var deliveredAt *string
	err := pool.QueryRow(ctx, `
		SELECT status, http_status, last_error, delivered_at
		FROM webhook_deliveries
		WHERE id = $1`, deliveryID).Scan(&deliveryStatus, &httpStatus, &lastError, &deliveredAt)
	if err != nil {
		t.Fatalf("query replayed webhook delivery: %v", err)
	}
	if deliveryStatus != "pending" {
		t.Fatalf("expected delivery status pending, got %s", deliveryStatus)
	}
	if httpStatus != nil || lastError != nil || deliveredAt != nil {
		t.Fatalf("expected delivery state to be reset, got http_status=%v last_error=%v delivered_at=%v", httpStatus, lastError, deliveredAt)
	}

	assertOutboxStatus(t, ctx, pool, outboxEventID, "pending")
}

func TestReplayOutboxEndpointsRequeueExpectedRows(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	outboxEventID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "processed")
	insertTestOutboxEvent(t, ctx, pool, 1, "organization.updated", "processed")

	resp := performJSONRequest(t, router, http.MethodPost, "/outbox-events/"+itoa(outboxEventID)+"/replay", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("replay outbox event returned status %d: %s", resp.Code, resp.Body.String())
	}
	assertOutboxStatus(t, ctx, pool, outboxEventID, "pending")

	resp = performJSONRequest(t, router, http.MethodPost, "/outbox-events/replay", map[string]interface{}{
		"event_type": "organization.updated",
		"status":     "processed",
		"limit":      10,
	}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("replay outbox events by filter returned status %d: %s", resp.Code, resp.Body.String())
	}

	var requeuedCount int64
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE tenant_id = 1 AND event_type = 'organization.updated' AND status = 'pending'`).Scan(&requeuedCount); err != nil {
		t.Fatalf("query requeued outbox events: %v", err)
	}
	if requeuedCount != 1 {
		t.Fatalf("expected 1 requeued organization.updated outbox event, got %d", requeuedCount)
	}
}

func TestGetOutboxEventsSupportsFilters(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "failed")
	insertTestOutboxEvent(t, ctx, pool, 1, "organization.updated", "processed")

	resp := performJSONRequest(t, router, http.MethodGet, "/outbox-events?status=failed&event_type=organization.created&limit=10", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("list outbox events returned status %d: %s", resp.Code, resp.Body.String())
	}

	var events []map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &events); err != nil {
		t.Fatalf("decode outbox events response: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}
	if events[0]["event_type"] != "organization.created" {
		t.Fatalf("unexpected outbox event_type: %#v", events[0]["event_type"])
	}
	if events[0]["status"] != "failed" {
		t.Fatalf("unexpected outbox status: %#v", events[0]["status"])
	}
}

func TestGetOutboxEventByIDReturnsEvent(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	outboxEventID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "processed")

	resp := performJSONRequest(t, router, http.MethodGet, "/outbox-events/"+itoa(outboxEventID), nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("get outbox event returned status %d: %s", resp.Code, resp.Body.String())
	}

	var event map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &event); err != nil {
		t.Fatalf("decode outbox event response: %v", err)
	}
	if int64(event["id"].(float64)) != outboxEventID {
		t.Fatalf("unexpected outbox event id: %#v", event["id"])
	}
	if event["event_type"] != "organization.created" {
		t.Fatalf("unexpected outbox event_type: %#v", event["event_type"])
	}
}

func TestGetOutboxStatsReturnsOperationalSummary(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	failedID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "failed")
	pendingID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.updated", "pending")
	insertTestOutboxEvent(t, ctx, pool, 1, "note.created", "processed")

	if _, err := pool.Exec(ctx, `
		UPDATE outbox_events
		SET last_error = 'boom', next_attempt_at = NOW() + INTERVAL '5 minutes'
		WHERE id = $1`, failedID); err != nil {
		t.Fatalf("update failed outbox event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE outbox_events
		SET next_attempt_at = NOW() + INTERVAL '1 minute'
		WHERE id = $1`, pendingID); err != nil {
		t.Fatalf("update pending outbox event: %v", err)
	}

	resp := performJSONRequest(t, router, http.MethodGet, "/outbox-events/stats", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("get outbox stats returned status %d: %s", resp.Code, resp.Body.String())
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode outbox stats response: %v", err)
	}

	byStatus, ok := stats["by_status"].([]interface{})
	if !ok || len(byStatus) == 0 {
		t.Fatalf("expected by_status entries, got %#v", stats["by_status"])
	}
	byEventType, ok := stats["by_event_type"].([]interface{})
	if !ok || len(byEventType) == 0 {
		t.Fatalf("expected by_event_type entries, got %#v", stats["by_event_type"])
	}
	recentFailed, ok := stats["recent_failed"].([]interface{})
	if !ok || len(recentFailed) != 1 {
		t.Fatalf("expected 1 recent_failed entry, got %#v", stats["recent_failed"])
	}
	nextRetry, ok := stats["next_retry"].([]interface{})
	if !ok || len(nextRetry) == 0 {
		t.Fatalf("expected next_retry entries, got %#v", stats["next_retry"])
	}
}

func TestGetWebhookDeliveryStatsReturnsOperationalSummary(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	endpoint := createTestWebhookEndpoint(t, router, token)
	endpointID := int64(endpoint["id"].(float64))
	failedOutboxID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.created", "failed")
	deliveredOutboxID := insertTestOutboxEvent(t, ctx, pool, 1, "organization.updated", "processed")
	insertTestWebhookDelivery(t, ctx, pool, 1, endpointID, failedOutboxID, "failed", 2)
	deliveredID := insertTestWebhookDelivery(t, ctx, pool, 1, endpointID, deliveredOutboxID, "delivered", 1)

	if _, err := pool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET delivered_at = NOW(), http_status = 200, last_error = NULL, response_body = 'ok'
		WHERE id = $1`, deliveredID); err != nil {
		t.Fatalf("update delivered webhook delivery: %v", err)
	}

	resp := performJSONRequest(t, router, http.MethodGet, "/webhook-deliveries/stats", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("get webhook delivery stats returned status %d: %s", resp.Code, resp.Body.String())
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode webhook delivery stats response: %v", err)
	}

	byStatus, ok := stats["by_status"].([]interface{})
	if !ok || len(byStatus) == 0 {
		t.Fatalf("expected by_status entries, got %#v", stats["by_status"])
	}
	byEndpoint, ok := stats["by_endpoint"].([]interface{})
	if !ok || len(byEndpoint) == 0 {
		t.Fatalf("expected by_endpoint entries, got %#v", stats["by_endpoint"])
	}
	recentFailed, ok := stats["recent_failed"].([]interface{})
	if !ok || len(recentFailed) != 1 {
		t.Fatalf("expected 1 recent_failed entry, got %#v", stats["recent_failed"])
	}
	recentDelivered, ok := stats["recent_delivered"].([]interface{})
	if !ok || len(recentDelivered) != 1 {
		t.Fatalf("expected 1 recent_delivered entry, got %#v", stats["recent_delivered"])
	}
}

func TestAdminTenantAndUsersLifecycle(t *testing.T) {
	ctx := context.Background()
	pool, router, token := setupIntegrationApp(t, ctx)
	defer pool.Close()

	resp := performJSONRequest(t, router, http.MethodGet, "/admin/tenant", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("get admin tenant returned status %d: %s", resp.Code, resp.Body.String())
	}

	updateTenantBody := map[string]interface{}{
		"name":   "Updated Integration Tenant",
		"slug":   "demo",
		"plan":   "pro",
		"status": "active",
	}
	resp = performJSONRequest(t, router, http.MethodPut, "/admin/tenant", updateTenantBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update admin tenant returned status %d: %s", resp.Code, resp.Body.String())
	}

	createUserBody := map[string]interface{}{
		"email":             "member@crmflow.local",
		"password":          "member123",
		"full_name":         "Team Member",
		"role":              "member",
		"user_status":       "active",
		"membership_status": "active",
	}
	resp = performJSONRequest(t, router, http.MethodPost, "/admin/users", createUserBody, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create admin user returned status %d: %s", resp.Code, resp.Body.String())
	}

	var createdUser map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &createdUser); err != nil {
		t.Fatalf("decode created admin user response: %v", err)
	}
	userID := int64(createdUser["id"].(float64))

	resp = performJSONRequest(t, router, http.MethodGet, "/admin/users", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("get admin users returned status %d: %s", resp.Code, resp.Body.String())
	}

	var users []map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode admin users response: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users in tenant, got %d", len(users))
	}

	updateUserBody := map[string]interface{}{
		"full_name":         "Team Member Updated",
		"role":              "admin",
		"user_status":       "active",
		"membership_status": "active",
	}
	resp = performJSONRequest(t, router, http.MethodPut, "/admin/users/"+itoa(userID), updateUserBody, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("update admin user returned status %d: %s", resp.Code, resp.Body.String())
	}

	assertAuditCount(t, ctx, pool, "updated", "tenant", 1)
	assertAuditCount(t, ctx, pool, "created", "user", 1)
	assertAuditCount(t, ctx, pool, "updated", "user", 1)
}

func resetIntegrationState(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			webhook_deliveries,
			webhook_subscriptions,
			webhook_endpoints,
			outbox_events,
			audit_log,
			notes,
			tasks,
			deal_stage_history,
			deals,
			people,
			organizations,
			tenant_memberships,
			users,
			tenants
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate integration tables: %v", err)
	}
}

func seedIntegrationUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, plan, status)
		VALUES (1, 'Integration Tenant', 'demo', 'starter', 'active')`,
	)
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, status)
		VALUES (1, 'admin@crmflow.local', $1, 'Integration Admin', 'active')`,
		testPasswordHash,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO tenant_memberships (tenant_id, user_id, role, status)
		VALUES (1, 1, 'owner', 'active')`,
	)
	if err != nil {
		t.Fatalf("seed integration user: %v", err)
	}
}

func loginTestUser(t *testing.T, router http.Handler) string {
	t.Helper()

	body := map[string]string{
		"email":       "admin@crmflow.local",
		"password":    "admin123",
		"tenant_slug": "demo",
	}

	resp := performJSONRequest(t, router, http.MethodPost, "/login", body, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("login returned status %d: %s", resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	token, ok := payload["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login response missing token: %#v", payload)
	}

	return token
}

func setupIntegrationApp(t *testing.T, ctx context.Context) (*pgxpool.Pool, http.Handler, string) {
	t.Helper()

	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = defaultDatabaseURL()
	}

	pool, err := initDBPool(ctx, testDBURL)
	if err != nil {
		t.Skipf("skipping integration test; database unavailable: %v", err)
	}

	prevPool := dbPool
	prevSecret := jwtSecret
	dbPool = pool
	jwtSecret = []byte("test-secret")
	t.Cleanup(func() {
		dbPool = prevPool
		jwtSecret = prevSecret
	})

	resetIntegrationState(t, ctx, pool)
	seedIntegrationUser(t, ctx, pool)
	router := newRouter()
	token := loginTestUser(t, router)
	return pool, router, token
}

func createTestOrganization(t *testing.T, router http.Handler, token string) map[string]interface{} {
	t.Helper()

	body := map[string]string{
		"name":     "Integration Org",
		"domain":   "integration.test",
		"industry": "QA",
	}

	resp := performJSONRequest(t, router, http.MethodPost, "/organizations", body, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create organization returned status %d: %s", resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode organization response: %v", err)
	}

	return payload
}

func createTestWebhookEndpoint(t *testing.T, router http.Handler, token string) map[string]interface{} {
	t.Helper()

	body := map[string]string{
		"name":           "Primary Endpoint",
		"target_url":     "https://example.test/webhook",
		"signing_secret": "super-secret",
	}

	resp := performJSONRequest(t, router, http.MethodPost, "/webhook-endpoints", body, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create webhook endpoint returned status %d: %s", resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode webhook endpoint response: %v", err)
	}

	return payload
}

func createTestWebhookSubscription(t *testing.T, router http.Handler, token string, endpointID int64, eventType string) int64 {
	t.Helper()

	body := map[string]interface{}{
		"webhook_endpoint_id": endpointID,
		"event_type":          eventType,
	}

	resp := performJSONRequest(t, router, http.MethodPost, "/webhook-subscriptions", body, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create webhook subscription returned status %d: %s", resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode webhook subscription response: %v", err)
	}

	return int64(payload["id"].(float64))
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}

func insertTestOutboxEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID int64, eventType, status string) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO outbox_events (tenant_id, event_type, payload, status)
		VALUES ($1, $2, '{}'::jsonb, $3)
		RETURNING id`,
		tenantID, eventType, status,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert outbox event: %v", err)
	}

	return id
}

func insertTestWebhookDelivery(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, endpointID, outboxEventID int64, status string, attemptCount int) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO webhook_deliveries (
			tenant_id, webhook_endpoint_id, outbox_event_id, status, attempt_count, http_status, last_error, response_body, delivered_at
		)
		VALUES ($1, $2, $3, $4, $5, 500, 'delivery failed', 'upstream error', NOW())
		RETURNING id`,
		tenantID, endpointID, outboxEventID, status, attemptCount,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert webhook delivery: %v", err)
	}

	return id
}

func assertOutboxCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventType string, want int) {
	t.Helper()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE tenant_id = 1 AND event_type = $1`, eventType).Scan(&count)
	if err != nil {
		t.Fatalf("query outbox_events for %s: %v", eventType, err)
	}
	if count != want {
		t.Fatalf("expected %d outbox events for %s, got %d", want, eventType, count)
	}
}

func assertAuditCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, action, entityType string, want int) {
	t.Helper()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM audit_log
		WHERE tenant_id = 1 AND action = $1 AND entity_type = $2`, action, entityType).Scan(&count)
	if err != nil {
		t.Fatalf("query audit_log for %s/%s: %v", action, entityType, err)
	}
	if count != want {
		t.Fatalf("expected %d audit entries for %s/%s, got %d", want, action, entityType, count)
	}
}

func assertOutboxStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, outboxEventID int64, want string) {
	t.Helper()

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM outbox_events WHERE id = $1`, outboxEventID).Scan(&status); err != nil {
		t.Fatalf("query outbox status for %d: %v", outboxEventID, err)
	}
	if status != want {
		t.Fatalf("expected outbox event %d status %s, got %s", outboxEventID, want, status)
	}
}

func performJSONRequest(t *testing.T, router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
