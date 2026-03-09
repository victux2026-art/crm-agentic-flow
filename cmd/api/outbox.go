package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"victux.local/domain"
)

func enqueueOutboxEvent(ctx context.Context, tenantID *int64, eventType string, payload interface{}) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling outbox payload for %s: %v\n", eventType, err)
		return
	}

	_, err = dbPool.Exec(ctx, `
		INSERT INTO outbox_events (tenant_id, event_type, payload, status, next_attempt_at)
		VALUES ($1, $2, $3::jsonb, 'pending', NOW())`,
		tenantID, eventType, payloadJSON,
	)
	if err != nil {
		log.Printf("Error enqueueing outbox event %s: %v\n", eventType, err)
	}
}

func enqueueLegacyEvent(ctx context.Context, eventType string, payload interface{}) {
	eventPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling legacy event payload for %s: %v\n", eventType, err)
		return
	}

	event := domain.Event{
		ID:        fmt.Sprintf("evt-%s-%d", eventType, time.Now().UnixNano()),
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   eventPayload,
		Retries:   0,
	}

	enqueueOutboxEvent(ctx, nil, eventType, event)
}
