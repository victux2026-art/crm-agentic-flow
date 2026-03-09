package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"victux.local/domain"

	chi "github.com/go-chi/chi/v5"
	pgx "github.com/jackc/pgx/v5"
)

// Legacy compatibility handlers below remain active only while clients migrate
// from persons/flows/activities to the MVP tenant-scoped model.
func createPerson(w http.ResponseWriter, r *http.Request) {
	var p domain.Person
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO persons (first_name, last_name, email, phone_number, company_name, job_title, linkedin_profile, status, metadata, embedding) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(context.Background(), query,
		p.FirstName, p.LastName, p.Email, p.PhoneNumber, p.CompanyName,
		p.JobTitle, p.LinkedinProfile, p.Status, p.Metadata, p.Embedding,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting person: %v\n", err)
		http.Error(w, "Could not create person", http.StatusInternalServerError)
		return
	}

	enqueueLegacyEvent(r.Context(), "person.created", p)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func getPersons(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(context.Background(), "SELECT id, first_name, last_name, email, phone_number, company_name, job_title, linkedin_profile, status, created_at, updated_at, metadata, embedding FROM persons ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error querying persons: %v\n", err)
		http.Error(w, "Could not fetch persons", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var persons []domain.Person
	for rows.Next() {
		var p domain.Person
		err := rows.Scan(
			&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.PhoneNumber, &p.CompanyName,
			&p.JobTitle, &p.LinkedinProfile, &p.Status, &p.CreatedAt, &p.UpdatedAt,
			&p.Metadata, &p.Embedding,
		)
		if err != nil {
			log.Printf("Error scanning person row: %v\n", err)
			http.Error(w, "Could not fetch persons", http.StatusInternalServerError)
			return
		}
		persons = append(persons, p)
	}

	if rows.Err() != nil {
		log.Printf("Error after iterating persons rows: %v\n", rows.Err())
		http.Error(w, "Could not fetch persons", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(persons)
}

func getPersonByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	var p domain.Person
	query := `SELECT id, first_name, last_name, email, phone_number, company_name, job_title, linkedin_profile, status, created_at, updated_at, metadata, embedding FROM persons WHERE id = $1`

	err = dbPool.QueryRow(context.Background(), query, id).Scan(
		&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.PhoneNumber, &p.CompanyName,
		&p.JobTitle, &p.LinkedinProfile, &p.Status, &p.CreatedAt, &p.UpdatedAt,
		&p.Metadata, &p.Embedding,
	)

	if err == pgx.ErrNoRows {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying person by ID: %v\n", err)
		http.Error(w, "Could not fetch person", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func updatePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	var p domain.Person
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `UPDATE persons SET first_name = $1, last_name = $2, email = $3, phone_number = $4, company_name = $5, job_title = $6, linkedin_profile = $7, status = $8, metadata = $9, embedding = $10, updated_at = CURRENT_TIMESTAMP WHERE id = $11 RETURNING updated_at`

	err = dbPool.QueryRow(context.Background(), query,
		p.FirstName, p.LastName, p.Email, p.PhoneNumber, p.CompanyName,
		p.JobTitle, p.LinkedinProfile, p.Status, p.Metadata, p.Embedding,
		id,
	).Scan(&p.UpdatedAt)

	if err == pgx.ErrNoRows {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating person: %v\n", err)
		http.Error(w, "Could not update person", http.StatusInternalServerError)
		return
	}

	p.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func deletePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(context.Background(), "DELETE FROM persons WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting person: %v\n", err)
		http.Error(w, "Could not delete person", http.StatusInternalServerError)
		return
	}

	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createFlow(w http.ResponseWriter, r *http.Request) {
	var f domain.Flow
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO flows (name, person_id, company_name, status, value, currency, expected_close_date, actual_close_date, priority, health_score, metadata, embedding) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(context.Background(), query,
		f.Name, f.PersonID, f.CompanyName, f.Status, f.Value, f.Currency,
		f.ExpectedCloseDate, f.ActualCloseDate, f.Priority, f.HealthScore, f.Metadata, f.Embedding,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting flow: %v\n", err)
		http.Error(w, "Could not create flow", http.StatusInternalServerError)
		return
	}

	enqueueLegacyEvent(r.Context(), "flow.created", f)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(f)
}

func getFlows(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(context.Background(), "SELECT id, name, person_id, company_name, status, value, currency, expected_close_date, actual_close_date, priority, health_score, created_at, updated_at, metadata, embedding FROM flows ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error querying flows: %v\n", err)
		http.Error(w, "Could not fetch flows", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var flows []domain.Flow
	for rows.Next() {
		var f domain.Flow
		err := rows.Scan(
			&f.ID, &f.Name, &f.PersonID, &f.CompanyName, &f.Status, &f.Value, &f.Currency,
			&f.ExpectedCloseDate, &f.ActualCloseDate, &f.Priority, &f.HealthScore, &f.CreatedAt, &f.UpdatedAt,
			&f.Metadata, &f.Embedding,
		)
		if err != nil {
			log.Printf("Error scanning flow row: %v\n", err)
			http.Error(w, "Could not fetch flows", http.StatusInternalServerError)
			return
		}
		flows = append(flows, f)
	}

	if rows.Err() != nil {
		log.Printf("Error after iterating flows rows: %v\n", rows.Err())
		http.Error(w, "Could not fetch flows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flows)
}

func getFlowByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid flow ID", http.StatusBadRequest)
		return
	}

	var f domain.Flow
	query := `SELECT id, name, person_id, company_name, status, value, currency, expected_close_date, actual_close_date, priority, health_score, created_at, updated_at, metadata, embedding FROM flows WHERE id = $1`

	err = dbPool.QueryRow(context.Background(), query, id).Scan(
		&f.ID, &f.Name, &f.PersonID, &f.CompanyName, &f.Status, &f.Value, &f.Currency,
		&f.ExpectedCloseDate, &f.ActualCloseDate, &f.Priority, &f.HealthScore, &f.CreatedAt, &f.UpdatedAt,
		&f.Metadata, &f.Embedding,
	)

	if err == pgx.ErrNoRows {
		http.Error(w, "Flow not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying flow by ID: %v\n", err)
		http.Error(w, "Could not fetch flow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f)
}

func updateFlow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid flow ID", http.StatusBadRequest)
		return
	}

	var f domain.Flow
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `UPDATE flows SET name = $1, person_id = $2, company_name = $3, status = $4, value = $5, currency = $6, expected_close_date = $7, actual_close_date = $8, priority = $9, health_score = $10, metadata = $11, embedding = $12, updated_at = CURRENT_TIMESTAMP WHERE id = $13 RETURNING updated_at`

	err = dbPool.QueryRow(context.Background(), query,
		f.Name, f.PersonID, f.CompanyName, f.Status, f.Value, f.Currency,
		f.ExpectedCloseDate, f.ActualCloseDate, f.Priority, f.HealthScore, f.Metadata, f.Embedding,
		id,
	).Scan(&f.UpdatedAt)

	if err == pgx.ErrNoRows {
		http.Error(w, "Flow not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating flow: %v\n", err)
		http.Error(w, "Could not update flow", http.StatusInternalServerError)
		return
	}

	f.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f)
}

func deleteFlow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid flow ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(context.Background(), "DELETE FROM flows WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting flow: %v\n", err)
		http.Error(w, "Could not delete flow", http.StatusInternalServerError)
		return
	}

	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Flow not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createActivity(w http.ResponseWriter, r *http.Request) {
	var a domain.Activity
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO activities (flow_id, person_id, type, subject, description, due_date, completed_at, status, metadata, embedding) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at, updated_at`

	err := dbPool.QueryRow(context.Background(), query,
		a.FlowID, a.PersonID, a.Type, a.Subject, a.Description, a.DueDate,
		a.CompletedAt, a.Status, a.Metadata, a.Embedding,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting activity: %v\n", err)
		http.Error(w, "Could not create activity", http.StatusInternalServerError)
		return
	}

	enqueueLegacyEvent(r.Context(), fmt.Sprintf("activity.%s", a.Type), a)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func getActivities(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(context.Background(), "SELECT id, flow_id, person_id, type, subject, description, due_date, completed_at, status, created_at, updated_at, metadata, embedding FROM activities ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error querying activities: %v\n", err)
		http.Error(w, "Could not fetch activities", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var activities []domain.Activity
	for rows.Next() {
		var a domain.Activity
		err := rows.Scan(
			&a.ID, &a.FlowID, &a.PersonID, &a.Type, &a.Subject, &a.Description, &a.DueDate,
			&a.CompletedAt, &a.Status, &a.CreatedAt, &a.UpdatedAt, &a.Metadata, &a.Embedding,
		)
		if err != nil {
			log.Printf("Error scanning activity row: %v\n", err)
			http.Error(w, "Could not fetch activities", http.StatusInternalServerError)
			return
		}
		activities = append(activities, a)
	}

	if rows.Err() != nil {
		log.Printf("Error after iterating activities rows: %v\n", rows.Err())
		http.Error(w, "Could not fetch activities", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

func getActivityByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return
	}

	var a domain.Activity
	query := `SELECT id, flow_id, person_id, type, subject, description, due_date, completed_at, status, created_at, updated_at, metadata, embedding FROM activities WHERE id = $1`

	err = dbPool.QueryRow(context.Background(), query, id).Scan(
		&a.ID, &a.FlowID, &a.PersonID, &a.Type, &a.Subject, &a.Description, &a.DueDate,
		&a.CompletedAt, &a.Status, &a.CreatedAt, &a.UpdatedAt, &a.Metadata, &a.Embedding,
	)

	if err == pgx.ErrNoRows {
		http.Error(w, "Activity not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying activity by ID: %v\n", err)
		http.Error(w, "Could not fetch activity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func updateActivity(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return
	}

	var a domain.Activity
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `UPDATE activities SET flow_id = $1, person_id = $2, type = $3, subject = $4, description = $5, due_date = $6, completed_at = $7, status = $8, metadata = $9, embedding = $10, updated_at = CURRENT_TIMESTAMP WHERE id = $11 RETURNING updated_at`

	err = dbPool.QueryRow(context.Background(), query,
		a.FlowID, a.PersonID, a.Type, a.Subject, a.Description, a.DueDate,
		a.CompletedAt, a.Status, a.Metadata, a.Embedding,
		id,
	).Scan(&a.UpdatedAt)

	if err == pgx.ErrNoRows {
		http.Error(w, "Activity not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating activity: %v\n", err)
		http.Error(w, "Could not update activity", http.StatusInternalServerError)
		return
	}

	a.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func deleteActivity(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return
	}

	commandTag, err := dbPool.Exec(context.Background(), "DELETE FROM activities WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting activity: %v\n", err)
		http.Error(w, "Could not delete activity", http.StatusInternalServerError)
		return
	}

	if commandTag.RowsAffected() == 0 {
		http.Error(w, "Activity not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
