package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
)

func writeAuditLog(ctx context.Context, r *http.Request, tenantID int64, action, entityType string, entityID int64, changes interface{}) {
	claims, _ := currentClaims(r)

	var actorID *int64
	if claims != nil {
		actorID = &claims.UserID
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		log.Printf("Error marshalling audit log changes for %s %s: %v\n", entityType, action, err)
		return
	}

	var ipAddress *string
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		ipAddress = &host
	}

	_, err = dbPool.Exec(ctx, `
		INSERT INTO audit_log (tenant_id, actor_type, actor_id, action, entity_type, entity_id, changes_json, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)`,
		tenantID,
		"user",
		actorID,
		action,
		entityType,
		entityID,
		changesJSON,
		ipAddress,
		r.UserAgent(),
	)
	if err != nil {
		log.Printf("Error writing audit log for %s %s: %v\n", entityType, action, err)
	}
}
