ALTER TABLE webhook_deliveries
ADD COLUMN IF NOT EXISTS outbox_event_id BIGINT REFERENCES outbox_events(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS idx_webhook_deliveries_endpoint_outbox;

CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint_outbox
ON webhook_deliveries (webhook_endpoint_id, outbox_event_id);
