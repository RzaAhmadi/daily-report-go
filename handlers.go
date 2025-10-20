package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// Authentication handlers
func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/login.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var user User
	var hashedPassword string
	err := db.QueryRow("SELECT id, username, full_name, role, password FROM users WHERE username = $1",
		credentials.Username).Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &hashedPassword)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(credentials.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["last_activity"] = time.Now().Unix()
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["user_id"] = nil
	session.Options.MaxAge = -1
	session.Save(r, w)
	w.WriteHeader(http.StatusOK)
}

func checkAuthHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		fmt.Printf("Session error: %v\n", err)
		http.Error(w, "Session error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Check if session has expired
	lastActivity, ok := session.Values["last_activity"].(int64)
	if !ok || time.Now().Unix()-lastActivity > int64(5*time.Minute.Seconds()) {
		// Session expired
		session.Options.MaxAge = -1
		session.Save(r, w)
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}
	
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		fmt.Println("User not authenticated")
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Update last activity
	session.Values["last_activity"] = time.Now().Unix()
	session.Save(r, w)

	var user User
	err = db.QueryRow("SELECT id, username, full_name, role FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Username, &user.FullName, &user.Role)
	if err != nil {
		fmt.Printf("Database error fetching user: %v\n", err)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		
		// Check if session has expired
		lastActivity, ok := session.Values["last_activity"].(int64)
		if !ok || time.Now().Unix()-lastActivity > int64(5*time.Minute.Seconds()) {
			// Session expired
			session.Options.MaxAge = -1
			session.Save(r, w)
			http.Error(w, "Session expired", http.StatusUnauthorized)
			return
		}
		
		if _, ok := session.Values["user_id"]; !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		
		// Update last activity
		session.Values["last_activity"] = time.Now().Unix()
		session.Save(r, w)
		
		next(w, r)
	}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		
		// Check if session has expired
		lastActivity, ok := session.Values["last_activity"].(int64)
		if !ok || time.Now().Unix()-lastActivity > int64(5*time.Minute.Seconds()) {
			// Session expired
			session.Options.MaxAge = -1
			session.Save(r, w)
			http.Error(w, "Session expired", http.StatusUnauthorized)
			return
		}
		
		role, ok := session.Values["role"].(string)
		if !ok || role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}
		
		// Update last activity
		session.Values["last_activity"] = time.Now().Unix()
		session.Save(r, w)
		
		next(w, r)
	}
}

// User handlers
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, username, full_name, role FROM users ORDER BY full_name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		rows.Scan(&user.ID, &user.Username, &user.FullName, &user.Role)
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	var id int
	err = db.QueryRow("INSERT INTO users (username, password, full_name, role) VALUES ($1, $2, $3, $4) RETURNING id",
		user.Username, string(hashedPassword), user.FullName, user.Role).Scan(&id)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var user struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
		Password string `json:"password,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		_, err = db.Exec("UPDATE users SET username = $1, full_name = $2, role = $3, password = $4 WHERE id = $5",
			user.Username, user.FullName, user.Role, string(hashedPassword), id)
	} else {
		_, err := db.Exec("UPDATE users SET username = $1, full_name = $2, role = $3 WHERE id = $4",
			user.Username, user.FullName, user.Role, id)
		if err != nil {
			http.Error(w, "Error updating user", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	_, err := db.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Shift hours handlers
func getShiftHoursHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, start_time, end_time FROM shift_hours ORDER BY start_time")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var shifts []ShiftHours
	for rows.Next() {
		var shift ShiftHours
		rows.Scan(&shift.ID, &shift.Name, &shift.StartTime, &shift.EndTime)
		shifts = append(shifts, shift)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shifts)
}

func createShiftHoursHandler(w http.ResponseWriter, r *http.Request) {
	var shift ShiftHours
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var id int
	err := db.QueryRow("INSERT INTO shift_hours (name, start_time, end_time) VALUES ($1, $2, $3) RETURNING id",
		shift.Name, shift.StartTime, shift.EndTime).Scan(&id)
	if err != nil {
		http.Error(w, "Error creating shift hours", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func updateShiftHoursHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var shift ShiftHours
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE shift_hours SET name = $1, start_time = $2, end_time = $3 WHERE id = $4",
		shift.Name, shift.StartTime, shift.EndTime, id)
	if err != nil {
		http.Error(w, "Error updating shift hours", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteShiftHoursHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	_, err := db.Exec("DELETE FROM shift_hours WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting shift hours", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Event titles handlers
func getEventTitlesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title FROM event_titles ORDER BY title")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var titles []EventTitle
	for rows.Next() {
		var title EventTitle
		rows.Scan(&title.ID, &title.Title)
		titles = append(titles, title)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(titles)
}

func createEventTitleHandler(w http.ResponseWriter, r *http.Request) {
	var title EventTitle
	if err := json.NewDecoder(r.Body).Decode(&title); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var id int
	err := db.QueryRow("INSERT INTO event_titles (title) VALUES ($1) RETURNING id", title.Title).Scan(&id)
	if err != nil {
		http.Error(w, "Error creating event title", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func updateEventTitleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var title EventTitle
	if err := json.NewDecoder(r.Body).Decode(&title); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE event_titles SET title = $1 WHERE id = $2", title.Title, id)
	if err != nil {
		http.Error(w, "Error updating event title", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteEventTitleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	_, err := db.Exec("DELETE FROM event_titles WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting event title", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Report handlers
func getReportsHandler(w http.ResponseWriter, r *http.Request) {
	dateFilter := r.URL.Query().Get("date")

	// Log the request
	fmt.Println("Received request for /api/reports")

	query := `
        SELECT dr.id, dr.report_date, dr.health_power_sources, dr.health_humidity_temp, 
               dr.health_fire_system, dr.created_at,
               u.id, u.username, u.full_name, u.role,
               sh.id, sh.name, sh.start_time, sh.end_time
        FROM daily_reports dr
        JOIN users u ON dr.created_by = u.id
        LEFT JOIN shift_hours sh ON dr.shift_hours_id = sh.id
    `

	args := []interface{}{}
	if dateFilter != "" {
		query += " WHERE dr.report_date = $1"
		args = append(args, dateFilter)
	}

	query += " ORDER BY dr.report_date DESC, dr.created_at DESC"

	// Log the query
	fmt.Printf("Executing query: %s with args: %v\n", query, args)

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
		http.Error(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reports []DailyReport
	for rows.Next() {
		var report DailyReport
		var shiftID sql.NullInt64
		var shiftName, shiftStart, shiftEnd sql.NullString
		var createdByUser User

		err := rows.Scan(&report.ID, &report.ReportDate, &report.HealthPowerSources,
			&report.HealthHumidityTemp, &report.HealthFireSystem, &report.CreatedAt,
			&createdByUser.ID, &createdByUser.Username, &createdByUser.FullName, &createdByUser.Role,
			&shiftID, &shiftName, &shiftStart, &shiftEnd)
		if err != nil {
			fmt.Printf("Row scanning error: %v\n", err)
			http.Error(w, "Row scanning error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		report.CreatedBy = createdByUser

		if shiftID.Valid {
			report.ShiftHours = &ShiftHours{
				ID:        int(shiftID.Int64),
				Name:      shiftName.String,
				StartTime: shiftStart.String,
				EndTime:   shiftEnd.String,
			}
		}

		// Load shift managers
		managerRows, err := db.Query(`
            SELECT u.id, u.username, u.full_name, u.role 
            FROM users u 
            JOIN report_shift_managers rsm ON u.id = rsm.user_id 
            WHERE rsm.report_id = $1`, report.ID)
		if err != nil {
			fmt.Printf("Error loading shift managers: %v\n", err)
			http.Error(w, "Error loading shift managers: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for managerRows.Next() {
			var manager User
			err := managerRows.Scan(&manager.ID, &manager.Username, &manager.FullName, &manager.Role)
			if err != nil {
				managerRows.Close()
				fmt.Printf("Error scanning shift manager: %v\n", err)
				http.Error(w, "Error scanning shift manager: "+err.Error(), http.StatusInternalServerError)
				return
			}
			report.ShiftManagers = append(report.ShiftManagers, manager)
		}
		managerRows.Close()

		// Load event titles
		titleRows, err := db.Query(`
            SELECT et.id, et.title 
            FROM event_titles et 
            JOIN report_event_titles ret ON et.id = ret.event_title_id 
            WHERE ret.report_id = $1`, report.ID)
		if err != nil {
			fmt.Printf("Error loading event titles: %v\n", err)
			http.Error(w, "Error loading event titles: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for titleRows.Next() {
			var title EventTitle
			err := titleRows.Scan(&title.ID, &title.Title)
			if err != nil {
				titleRows.Close()
				fmt.Printf("Error scanning event title: %v\n", err)
				http.Error(w, "Error scanning event title: "+err.Error(), http.StatusInternalServerError)
				return
			}
			report.EventTitles = append(report.EventTitles, title)
		}
		titleRows.Close()

		// Load Part 3 events
		part3Rows, err := db.Query(`
            SELECT id, event_summary, trigger_info, start_time, end_time, rca_number 
            FROM report_events_part3 
            WHERE report_id = $1 
            ORDER BY id`, report.ID)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer part3Rows.Close()

		for part3Rows.Next() {
			var event EventPart3
			var startTime, endTime sql.NullString
			err := part3Rows.Scan(&event.ID, &event.EventSummary, &event.Trigger, &startTime, &endTime, &event.RCANumber)
			if err != nil {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if startTime.Valid {
				// Extract only the time portion from TIMESTAMP values
				if len(startTime.String) >= 19 { // TIMESTAMP format: "YYYY-MM-DD HH:MM:SS"
					event.StartTime = startTime.String[11:16] // Extract HH:MM
				} else {
					event.StartTime = startTime.String
				}
			}
			if endTime.Valid {
				// Extract only the time portion from TIMESTAMP values
				if len(endTime.String) >= 19 { // TIMESTAMP format: "YYYY-MM-DD HH:MM:SS"
					event.EndTime = endTime.String[11:16] // Extract HH:MM
				} else {
					event.EndTime = endTime.String
				}
			}
			report.EventsPart3 = append(report.EventsPart3, event)
		}

		// Check for errors from iterating over part 3 rows
		if err = part3Rows.Err(); err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Load Part 4 events
		part4Rows, err := db.Query(`
            SELECT id, event_summary, trigger_info, start_time, end_time 
            FROM report_events_part4 
            WHERE report_id = $1 
            ORDER BY id`, report.ID)
		if err != nil {
			fmt.Printf("Error loading Part 4 events: %v\n", err)
			http.Error(w, "Error loading Part 4 events: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for part4Rows.Next() {
			var event EventPart4
			var startTime, endTime sql.NullString
			err := part4Rows.Scan(&event.ID, &event.EventSummary, &event.Trigger, &startTime, &endTime)
			if err != nil {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if startTime.Valid {
				// Extract only the time portion from TIMESTAMP values
				if len(startTime.String) >= 19 { // TIMESTAMP format: "YYYY-MM-DD HH:MM:SS"
					event.StartTime = startTime.String[11:16] // Extract HH:MM
				} else {
					event.StartTime = startTime.String
				}
			}
			if endTime.Valid {
				// Extract only the time portion from TIMESTAMP values
				if len(endTime.String) >= 19 { // TIMESTAMP format: "YYYY-MM-DD HH:MM:SS"
					event.EndTime = endTime.String[11:16] // Extract HH:MM
				} else {
					event.EndTime = endTime.String
				}
			}
			report.EventsPart4 = append(report.EventsPart4, event)
		}
		part4Rows.Close()

		reports = append(reports, report)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		fmt.Printf("Rows iteration error: %v\n", err)
		http.Error(w, "Rows iteration error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Returning %d reports\n", len(reports))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func getReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	var report DailyReport
	var shiftID sql.NullInt64
	var shiftName, shiftStart, shiftEnd sql.NullString
	var createdByUser User

	err = db.QueryRow(`
        SELECT dr.id, dr.report_date, dr.health_power_sources, dr.health_humidity_temp, 
               dr.health_fire_system, dr.created_at,
               u.id, u.username, u.full_name, u.role,
               sh.id, sh.name, sh.start_time, sh.end_time
        FROM daily_reports dr
        JOIN users u ON dr.created_by = u.id
        LEFT JOIN shift_hours sh ON dr.shift_hours_id = sh.id
        WHERE dr.id = $1`, id).Scan(
		&report.ID, &report.ReportDate, &report.HealthPowerSources,
		&report.HealthHumidityTemp, &report.HealthFireSystem, &report.CreatedAt,
		&createdByUser.ID, &createdByUser.Username, &createdByUser.FullName, &createdByUser.Role,
		&shiftID, &shiftName, &shiftStart, &shiftEnd)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Report not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	report.CreatedBy = createdByUser

	if shiftID.Valid {
		report.ShiftHours = &ShiftHours{
			ID:        int(shiftID.Int64),
			Name:      shiftName.String,
			StartTime: shiftStart.String,
			EndTime:   shiftEnd.String,
		}
	}

	// Load shift managers
	managerRows, err := db.Query(`
        SELECT u.id, u.username, u.full_name, u.role 
        FROM users u 
        JOIN report_shift_managers rsm ON u.id = rsm.user_id 
        WHERE rsm.report_id = $1`, report.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer managerRows.Close()

	for managerRows.Next() {
		var manager User
		err := managerRows.Scan(&manager.ID, &manager.Username, &manager.FullName, &manager.Role)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		report.ShiftManagers = append(report.ShiftManagers, manager)
	}

	// Check for errors from iterating over manager rows
	if err = managerRows.Err(); err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load event titles
	titleRows, err := db.Query(`
        SELECT et.id, et.title 
        FROM event_titles et 
        JOIN report_event_titles ret ON et.id = ret.event_title_id 
        WHERE ret.report_id = $1`, report.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer titleRows.Close()

	for titleRows.Next() {
		var title EventTitle
		err := titleRows.Scan(&title.ID, &title.Title)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		report.EventTitles = append(report.EventTitles, title)
	}

	// Check for errors from iterating over title rows
	if err = titleRows.Err(); err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load Part 3 events
	part3Rows, err := db.Query(`
        SELECT id, event_summary, trigger_info, start_time, end_time, rca_number 
        FROM report_events_part3 
        WHERE report_id = $1 
        ORDER BY id`, report.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer part3Rows.Close()

	for part3Rows.Next() {
		var event EventPart3
		var startTime, endTime sql.NullString
		err := part3Rows.Scan(&event.ID, &event.EventSummary, &event.Trigger, &startTime, &endTime, &event.RCANumber)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if startTime.Valid {
			event.StartTime = startTime.String
		}
		if endTime.Valid {
			event.EndTime = endTime.String
		}
		report.EventsPart3 = append(report.EventsPart3, event)
	}

	// Check for errors from iterating over part 3 rows
	if err = part3Rows.Err(); err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load Part 4 events
	part4Rows, err := db.Query(`
        SELECT id, event_summary, trigger_info, start_time, end_time 
        FROM report_events_part4 
        WHERE report_id = $1 
        ORDER BY id`, report.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer part4Rows.Close()

	for part4Rows.Next() {
		var event EventPart4
		var startTime, endTime sql.NullString
		err := part4Rows.Scan(&event.ID, &event.EventSummary, &event.Trigger, &startTime, &endTime)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if startTime.Valid {
			event.StartTime = startTime.String
		}
		if endTime.Valid {
			event.EndTime = endTime.String
		}
		report.EventsPart4 = append(report.EventsPart4, event)
	}

	// Check for errors from iterating over part 4 rows
	if err = part4Rows.Err(); err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func createReportHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID := session.Values["user_id"].(int)

	var reportData struct {
		ReportDate         string       `json:"report_date"`
		ShiftHoursID       *int         `json:"shift_hours_id"`
		ShiftManagerIDs    []int        `json:"shift_manager_ids"`
		EventTitleIDs      []int        `json:"event_title_ids"`
		HealthPowerSources bool         `json:"health_power_sources"`
		HealthHumidityTemp bool         `json:"health_humidity_temp"`
		HealthFireSystem   bool         `json:"health_fire_system"`
		EventsPart3        []EventPart3 `json:"events_part3"`
		EventsPart4        []EventPart4 `json:"events_part4"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reportData); err != nil {
		fmt.Printf("Error decoding report creation JSON: %v\n", err)
		http.Error(w, "Invalid JSON in report creation: "+err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var reportID int
	err = tx.QueryRow(`
        INSERT INTO daily_reports (report_date, shift_hours_id, health_power_sources, 
                                 health_humidity_temp, health_fire_system, created_by) 
        VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		reportData.ReportDate, reportData.ShiftHoursID, reportData.HealthPowerSources,
		reportData.HealthHumidityTemp, reportData.HealthFireSystem, userID).Scan(&reportID)

	if err != nil {
		http.Error(w, "Error creating report", http.StatusInternalServerError)
		return
	}

	// Insert shift managers
	for _, managerID := range reportData.ShiftManagerIDs {
		_, err = tx.Exec("INSERT INTO report_shift_managers (report_id, user_id) VALUES ($1, $2)",
			reportID, managerID)
		if err != nil {
			http.Error(w, "Error adding shift managers", http.StatusInternalServerError)
			return
		}
	}

	// Insert event titles
	for _, titleID := range reportData.EventTitleIDs {
		_, err = tx.Exec("INSERT INTO report_event_titles (report_id, event_title_id) VALUES ($1, $2)",
			reportID, titleID)
		if err != nil {
			http.Error(w, "Error adding event titles", http.StatusInternalServerError)
			return
		}
	}

	// Insert Part 3 events
	for _, event := range reportData.EventsPart3 {
		// Convert time-only values to TIMESTAMP format if needed
		startTime := event.StartTime
		endTime := event.EndTime
		
		// If we have time-only values (HH:MM format), convert them to TIMESTAMP
		if startTime != "" && len(startTime) <= 5 { // HH:MM format
			startTime = "1970-01-01 " + startTime + ":00"
		}
		if endTime != "" && len(endTime) <= 5 { // HH:MM format
			endTime = "1970-01-01 " + endTime + ":00"
		}
		
		_, err = tx.Exec(`
            INSERT INTO report_events_part3 (report_id, event_summary, trigger_info, start_time, end_time, rca_number) 
            VALUES ($1, $2, $3, $4, $5, $6)`,
			reportID, event.EventSummary, event.Trigger, startTime, endTime, event.RCANumber)
		if err != nil {
			fmt.Printf("Database error adding Part 3 events: %v\n", err)
			http.Error(w, "Error adding Part 3 events: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Insert Part 4 events
	for _, event := range reportData.EventsPart4 {
		// Convert time-only values to TIMESTAMP format if needed
		startTime := event.StartTime
		endTime := event.EndTime
		
		// If we have time-only values (HH:MM format), convert them to TIMESTAMP
		if startTime != "" && len(startTime) <= 5 { // HH:MM format
			startTime = "1970-01-01 " + startTime + ":00"
		}
		if endTime != "" && len(endTime) <= 5 { // HH:MM format
			endTime = "1970-01-01 " + endTime + ":00"
		}
		
		_, err = tx.Exec(`
            INSERT INTO report_events_part4 (report_id, event_summary, trigger_info, start_time, end_time) 
            VALUES ($1, $2, $3, $4, $5)`,
			reportID, event.EventSummary, event.Trigger, startTime, endTime)
		if err != nil {
			fmt.Printf("Database error adding Part 4 events: %v\n", err)
			http.Error(w, "Error adding Part 4 events: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tx.Commit()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": reportID})
}

func updateReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	var reportData struct {
		ReportDate         string       `json:"report_date"`
		ShiftHoursID       *int         `json:"shift_hours_id"`
		ShiftManagerIDs    []int        `json:"shift_manager_ids"`
		EventTitleIDs      []int        `json:"event_title_ids"`
		HealthPowerSources bool         `json:"health_power_sources"`
		HealthHumidityTemp bool         `json:"health_humidity_temp"`
		HealthFireSystem   bool         `json:"health_fire_system"`
		EventsPart3        []EventPart3 `json:"events_part3"`
		EventsPart4        []EventPart4 `json:"events_part4"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reportData); err != nil {
		fmt.Printf("Error decoding report update JSON: %v\n", err)
		http.Error(w, "Invalid JSON in report update: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Check if report exists and user has permission to update it
	var existingReport struct {
		ID        int `json:"id"`
		CreatedBy int `json:"created_by"`
	}

	err = db.QueryRow("SELECT id, created_by FROM daily_reports WHERE id = $1", id).Scan(&existingReport.ID, &existingReport.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Report not found", http.StatusNotFound)
		} else {
			fmt.Printf("Database error checking report existence: %v\n", err)
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get user info from session
	session, err := store.Get(r, "session")
	if err != nil {
		fmt.Printf("Session error: %v\n", err)
		http.Error(w, "Session error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userRole, ok := session.Values["role"].(string)
	if !ok {
		fmt.Printf("Session role error\n")
		http.Error(w, "Session error: Role not found in session", http.StatusInternalServerError)
		return
	}

	// Check if user can update this report (owner or admin)
	if existingReport.CreatedBy != userID && userRole != "admin" {
		fmt.Printf("Permission denied: User %d tried to update report %d created by %d\n", userID, id, existingReport.CreatedBy)
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Printf("Error beginning transaction: %v\n", err)
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Update main report
	_, err = tx.Exec(`
		UPDATE daily_reports SET 
			report_date = $1, 
			shift_hours_id = $2, 
			health_power_sources = $3, 
			health_humidity_temp = $4, 
			health_fire_system = $5
		WHERE id = $6`,
		reportData.ReportDate, reportData.ShiftHoursID, reportData.HealthPowerSources,
		reportData.HealthHumidityTemp, reportData.HealthFireSystem, id)

	if err != nil {
		fmt.Printf("Error updating daily_reports: %v\n", err)
		http.Error(w, "Failed to update report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete existing relationships
	_, err = tx.Exec("DELETE FROM report_shift_managers WHERE report_id = $1", id)
	if err != nil {
		fmt.Printf("Error deleting report_shift_managers: %v\n", err)
		http.Error(w, "Failed to update shift managers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM report_event_titles WHERE report_id = $1", id)
	if err != nil {
		fmt.Printf("Error deleting report_event_titles: %v\n", err)
		http.Error(w, "Failed to update event titles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM report_events_part3 WHERE report_id = $1", id)
	if err != nil {
		fmt.Printf("Error deleting report_events_part3: %v\n", err)
		http.Error(w, "Failed to update Part 3 events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM report_events_part4 WHERE report_id = $1", id)
	if err != nil {
		fmt.Printf("Error deleting report_events_part4: %v\n", err)
		http.Error(w, "Failed to update Part 4 events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-insert shift managers
	for _, managerID := range reportData.ShiftManagerIDs {
		_, err = tx.Exec("INSERT INTO report_shift_managers (report_id, user_id) VALUES ($1, $2)", id, managerID)
		if err != nil {
			fmt.Printf("Error inserting report_shift_managers: %v\n", err)
			http.Error(w, "Failed to update shift managers: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Re-insert event titles
	for _, titleID := range reportData.EventTitleIDs {
		_, err = tx.Exec("INSERT INTO report_event_titles (report_id, event_title_id) VALUES ($1, $2)", id, titleID)
		if err != nil {
			fmt.Printf("Error inserting report_event_titles: %v\n", err)
			http.Error(w, "Failed to update event titles: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Re-insert Part 3 events
	for _, event := range reportData.EventsPart3 {
		// Convert time-only values to TIMESTAMP format if needed
		startTime := event.StartTime
		endTime := event.EndTime
		
		// If we have time-only values (HH:MM format), convert them to TIMESTAMP
		if startTime != "" && len(startTime) <= 5 { // HH:MM format
			startTime = "1970-01-01 " + startTime + ":00"
		}
		if endTime != "" && len(endTime) <= 5 { // HH:MM format
			endTime = "1970-01-01 " + endTime + ":00"
		}
		
		_, err = tx.Exec(`
			INSERT INTO report_events_part3 (report_id, event_summary, trigger_info, start_time, end_time, rca_number) 
			VALUES ($1, $2, $3, $4, $5, $6)`,
			id, event.EventSummary, event.Trigger, startTime, endTime, event.RCANumber)

		if err != nil {
			fmt.Printf("Error inserting report_events_part3: %v\n", err)
			http.Error(w, "Failed to update Part 3 events: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Re-insert Part 4 events
	for _, event := range reportData.EventsPart4 {
		// Convert time-only values to TIMESTAMP format if needed
		startTime := event.StartTime
		endTime := event.EndTime
		
		// If we have time-only values (HH:MM format), convert them to TIMESTAMP
		if startTime != "" && len(startTime) <= 5 { // HH:MM format
			startTime = "1970-01-01 " + startTime + ":00"
		}
		if endTime != "" && len(endTime) <= 5 { // HH:MM format
			endTime = "1970-01-01 " + endTime + ":00"
		}
		
		_, err = tx.Exec(`
			INSERT INTO report_events_part4 (report_id, event_summary, trigger_info, start_time, end_time) 
			VALUES ($1, $2, $3, $4, $5)`,
			id, event.EventSummary, event.Trigger, startTime, endTime)

		if err != nil {
			fmt.Printf("Error inserting report_events_part4: %v\n", err)
			http.Error(w, "Failed to update Part 4 events: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Failed to save changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Report updated successfully"})
}

func deleteReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete related records first
	_, err = tx.Exec("DELETE FROM report_shift_managers WHERE report_id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting report shift managers", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM report_event_titles WHERE report_id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting report event titles", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM report_events_part3 WHERE report_id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting report events part 3", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM report_events_part4 WHERE report_id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting report events part 4", http.StatusInternalServerError)
		return
	}

	// Delete main report
	_, err = tx.Exec("DELETE FROM daily_reports WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Error deleting report", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	w.WriteHeader(http.StatusOK)
}