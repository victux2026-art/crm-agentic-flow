package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRetries      = 5
	pollInterval    = 2 * time.Second
	claimBatchSize  = 10
	processingState = "processing"
)

type outboxEvent struct {
	ID           int64
	TenantID     *int64
	EventType    string
	Payload      []byte
	Status       string
	AttemptCount int
}

type webhookTarget struct {
	EndpointID    int64
	TargetURL     string
	SigningSecret string
}

var dbPool *pgxpool.Pool
var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5440/crm_agentic_flow"
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v\n", err)
	}

	dbPool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Unable to connect to PostgreSQL: %v\n", err)
	}

	log.Println("Event processor connected to PostgreSQL")
	runProcessorLoop(ctx)
}

func runProcessorLoop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := processAvailableEvents(ctx); err != nil {
			log.Printf("Error processing outbox events: %v\n", err)
		}
		<-ticker.C
	}
}

func processAvailableEvents(ctx context.Context) error {
	events, err := claimOutboxEvents(ctx, claimBatchSize)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		if err := handleOutboxEvent(ctx, event); err != nil {
			log.Printf("Event %d (%s) failed: %v\n", event.ID, event.EventType, err)
			if markErr := markEventFailed(ctx, event.ID, event.AttemptCount, err.Error()); markErr != nil {
				log.Printf("Error updating failed event %d: %v\n", event.ID, markErr)
			}
			continue
		}

		if err := markEventProcessed(ctx, event.ID); err != nil {
			log.Printf("Error marking processed event %d: %v\n", event.ID, err)
		}
	}

	return nil
}

func claimOutboxEvents(ctx context.Context, limit int) ([]outboxEvent, error) {
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM outbox_events
			WHERE status = 'pending' AND next_attempt_at <= NOW()
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events oe
		SET status = $2,
		    attempt_count = oe.attempt_count + 1
		FROM claimed
		WHERE oe.id = claimed.id
		RETURNING oe.id, oe.tenant_id, oe.event_type, oe.payload::text, oe.status, oe.attempt_count`,
		limit, processingState,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []outboxEvent
	for rows.Next() {
		var event outboxEvent
		var payloadText string
		if err := rows.Scan(&event.ID, &event.TenantID, &event.EventType, &payloadText, &event.Status, &event.AttemptCount); err != nil {
			return nil, err
		}
		event.Payload = []byte(payloadText)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return events, nil
}

func handleOutboxEvent(_ context.Context, event outboxEvent) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	if event.TenantID != nil {
		if err := dispatchWebhookDeliveries(context.Background(), event, payload); err != nil {
			return err
		}
	}

	log.Printf(
		"Processed outbox event id=%d tenant=%v type=%s payload_keys=%d",
		event.ID,
		event.TenantID,
		event.EventType,
		len(payload),
	)

	return nil
}

func dispatchWebhookDeliveries(ctx context.Context, event outboxEvent, payload map[string]interface{}) error {
	if event.TenantID == nil {
		return nil
	}

	targets, err := loadWebhookTargets(ctx, *event.TenantID, event.EventType)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		return nil
	}

	bodyPayload := map[string]interface{}{
		"outbox_event_id": event.ID,
		"event_type":      event.EventType,
		"tenant_id":       event.TenantID,
		"payload":         payload,
	}

	requestBody, err := json.Marshal(bodyPayload)
	if err != nil {
		return err
	}

	for _, target := range targets {
		if err := deliverToTarget(ctx, event, target, requestBody); err != nil {
			return err
		}
	}

	return nil
}

func loadWebhookTargets(ctx context.Context, tenantID int64, eventType string) ([]webhookTarget, error) {
	rows, err := dbPool.Query(ctx, `
		SELECT we.id, we.target_url, we.signing_secret
		FROM webhook_subscriptions ws
		INNER JOIN webhook_endpoints we ON we.id = ws.webhook_endpoint_id
		WHERE ws.tenant_id = $1
		  AND ws.event_type = $2
		  AND ws.is_active = TRUE
		  AND we.status = 'active'`, tenantID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []webhookTarget
	for rows.Next() {
		var target webhookTarget
		if err := rows.Scan(&target.EndpointID, &target.TargetURL, &target.SigningSecret); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, rows.Err()
}

func deliverToTarget(ctx context.Context, event outboxEvent, target webhookTarget, requestBody []byte) error {
	headersJSON, signature := buildWebhookHeaders(event, target.SigningSecret)
	deliveryID, err := ensureWebhookDelivery(ctx, event, target.EndpointID, headersJSON, requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.TargetURL, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CRM-Event-Type", event.EventType)
	req.Header.Set("X-CRM-Outbox-Event-ID", fmt.Sprintf("%d", event.ID))
	req.Header.Set("X-CRM-Signature", signature)

	resp, err := httpClient.Do(req)
	if err != nil {
		return markWebhookDeliveryFailed(ctx, deliveryID, err.Error())
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return markWebhookDeliveryDelivered(ctx, deliveryID, resp.StatusCode, string(responseBody))
	}

	return markWebhookDeliveryFailedWithStatus(ctx, deliveryID, resp.StatusCode, string(responseBody))
}

func buildWebhookHeaders(event outboxEvent, signingSecret string) ([]byte, string) {
	headerMap := map[string]string{
		"Content-Type":         "application/json",
		"X-CRM-Event-Type":     event.EventType,
		"X-CRM-Outbox-Event-ID": fmt.Sprintf("%d", event.ID),
	}
	headerJSON, _ := json.Marshal(headerMap)

	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(fmt.Sprintf("%d:%s", event.ID, event.EventType)))
	signature := hex.EncodeToString(mac.Sum(nil))

	return headerJSON, signature
}

func ensureWebhookDelivery(ctx context.Context, event outboxEvent, endpointID int64, headersJSON []byte, requestBody []byte) (int64, error) {
	var deliveryID int64
	err := dbPool.QueryRow(ctx, `
		INSERT INTO webhook_deliveries (
			tenant_id, webhook_endpoint_id, outbox_event_id, status, attempt_count, request_headers, request_body, next_attempt_at
		)
		VALUES ($1, $2, $3, 'processing', 1, $4::jsonb, $5::jsonb, NOW())
		ON CONFLICT (webhook_endpoint_id, outbox_event_id)
		DO UPDATE SET
			status = 'processing',
			attempt_count = webhook_deliveries.attempt_count + 1,
			request_headers = EXCLUDED.request_headers,
			request_body = EXCLUDED.request_body,
			next_attempt_at = NOW()
		RETURNING id`,
		event.TenantID, endpointID, event.ID, headersJSON, requestBody,
	).Scan(&deliveryID)
	return deliveryID, err
}

func markWebhookDeliveryDelivered(ctx context.Context, deliveryID int64, httpStatus int, responseBody string) error {
	_, err := dbPool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'delivered',
		    http_status = $2,
		    response_body = $3,
		    last_error = NULL,
		    delivered_at = NOW()
		WHERE id = $1`, deliveryID, httpStatus, responseBody)
	return err
}

func markWebhookDeliveryFailed(ctx context.Context, deliveryID int64, lastError string) error {
	_, err := dbPool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'failed',
		    last_error = $2
		WHERE id = $1`, deliveryID, lastError)
	return err
}

func markWebhookDeliveryFailedWithStatus(ctx context.Context, deliveryID int64, httpStatus int, responseBody string) error {
	_, err := dbPool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'failed',
		    http_status = $2,
		    response_body = $3,
		    last_error = $4
		WHERE id = $1`, deliveryID, httpStatus, responseBody, fmt.Sprintf("webhook returned status %d", httpStatus))
	return err
}

func markEventProcessed(ctx context.Context, id int64) error {
	_, err := dbPool.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'processed',
		    processed_at = NOW(),
		    last_error = NULL
		WHERE id = $1`, id)
	return err
}

func markEventFailed(ctx context.Context, id int64, attemptCount int, lastError string) error {
	status := "pending"
	nextAttemptAt := time.Now().Add(backoffForAttempt(attemptCount))
	if attemptCount >= maxRetries {
		status = "failed"
	}

	_, err := dbPool.Exec(ctx, `
		UPDATE outbox_events
		SET status = $2,
		    last_error = $3,
		    next_attempt_at = $4
		WHERE id = $1`,
		id, status, lastError, nextAttemptAt,
	)
	return err
}

func backoffForAttempt(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 1 * time.Minute
	case 2:
		return 5 * time.Minute
	case 3:
		return 15 * time.Minute
	case 4:
		return 1 * time.Hour
	default:
		return 6 * time.Hour
	}
}
