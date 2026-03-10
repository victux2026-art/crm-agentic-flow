package main

import (
	"encoding/json"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID     int64  `json:"user_id"`
	TenantID   int64  `json:"tenant_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	TenantSlug string `json:"tenant_slug"`
	jwt.RegisteredClaims
}

type contextKey string

const authClaimsContextKey contextKey = "auth_claims"

func currentClaims(r *http.Request) (*Claims, bool) {
	claims, ok := r.Context().Value(authClaimsContextKey).(*Claims)
	return claims, ok
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func requireAdminClaims(w http.ResponseWriter, r *http.Request) (*Claims, bool) {
	claims, ok := currentClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	switch strings.ToLower(strings.TrimSpace(claims.Role)) {
	case "owner", "admin":
		return claims, true
	default:
		http.Error(w, "Forbidden", http.StatusForbidden)
		return nil, false
	}
}
