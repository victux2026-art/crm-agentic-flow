package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var dbPool *pgxpool.Pool

// JWT Secret Key - ¡IMPORTANTE: Usar una clave segura en producción!
var jwtSecret []byte

func main() {
	var err error
	dbPool, err = initDBPool(context.Background(), "")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	fmt.Println("Connected to PostgreSQL!")
	initJWTSecret()

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", newRouter()))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"service": "crm-agentic-flow-api",
	})
}

// authMiddleware verifica la validez del JWT
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[0:6] == "Bearer" {
			tokenString = authHeader[7:]
		}

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			ctx := context.WithValue(r.Context(), authClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		TenantSlug string `json:"tenant_slug"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var user struct {
		ID           int64
		Email        string
		PasswordHash string
	}

	userQuery := `SELECT id, email, password_hash FROM users WHERE email = $1 AND status = 'active'`
	err := dbPool.QueryRow(r.Context(), userQuery, credentials.Email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err == pgx.ErrNoRows {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Error loading user for login: %v\n", err)
		http.Error(w, "Could not complete login", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(credentials.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	var membership struct {
		TenantID   int64
		Role       string
		TenantSlug string
	}

	if credentials.TenantSlug != "" {
		membershipQuery := `
			SELECT tm.tenant_id, tm.role, t.slug
			FROM tenant_memberships tm
			INNER JOIN tenants t ON t.id = tm.tenant_id
			WHERE tm.user_id = $1 AND tm.status = 'active' AND t.status = 'active' AND t.slug = $2
			LIMIT 1`
		err = dbPool.QueryRow(r.Context(), membershipQuery, user.ID, credentials.TenantSlug).Scan(
			&membership.TenantID, &membership.Role, &membership.TenantSlug,
		)
	} else {
		membershipQuery := `
			SELECT tm.tenant_id, tm.role, t.slug
			FROM tenant_memberships tm
			INNER JOIN tenants t ON t.id = tm.tenant_id
			WHERE tm.user_id = $1 AND tm.status = 'active' AND t.status = 'active'
			ORDER BY tm.id ASC
			LIMIT 1`
		err = dbPool.QueryRow(r.Context(), membershipQuery, user.ID).Scan(
			&membership.TenantID, &membership.Role, &membership.TenantSlug,
		)
	}

	if err == pgx.ErrNoRows {
		http.Error(w, "No active tenant membership found", http.StatusForbidden)
		return
	} else if err != nil {
		log.Printf("Error loading tenant membership: %v\n", err)
		http.Error(w, "Could not complete login", http.StatusInternalServerError)
		return
	}

	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		UserID:     user.ID,
		TenantID:   membership.TenantID,
		Email:      user.Email,
		Role:       membership.Role,
		TenantSlug: membership.TenantSlug,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "crm-agentic-flow",
			Subject:   strconv.FormatInt(user.ID, 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Error signing token: %v\n", err)
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":       tokenString,
		"expires_at":  expirationTime.Format(time.RFC3339),
		"user_id":     user.ID,
		"email":       user.Email,
		"tenant_id":   membership.TenantID,
		"tenant_slug": membership.TenantSlug,
		"role":        membership.Role,
	})
}
