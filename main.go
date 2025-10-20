package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
)

var (
	db    *sql.DB
	store = sessions.NewCookieStore([]byte("your-secret-key-change-this"))
)

func main() {
	// Database connection
	var err error
	fmt.Println("Attempting to connect to database...")
	db, err = sql.Open("postgres", "host=localhost port=5432 user=postgres password=1 dbname=daily_reports sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	fmt.Println("Testing database connection...")
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	fmt.Println("Database connection successful!")

	// Configure session store with proper cookie settings
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = false // Set to true in production with HTTPS
	store.Options.SameSite = http.SameSiteLaxMode // Use Lax instead of None for better compatibility
	
	// We'll handle expiration ourselves in the handlers
	// Note: SameSite=None requires Secure=true, which is why we're using Lax instead

	r := mux.NewRouter()

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Routes
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/api/users", getUsersHandler).Methods("GET")
	r.HandleFunc("/api/users", createUserHandler).Methods("POST")
	r.HandleFunc("/api/users/{id}", updateUserHandler).Methods("PUT")
	r.HandleFunc("/api/users/{id}", deleteUserHandler).Methods("DELETE")
	r.HandleFunc("/api/shift-hours", getShiftHoursHandler).Methods("GET")
	r.HandleFunc("/api/shift-hours", createShiftHoursHandler).Methods("POST")
	r.HandleFunc("/api/shift-hours/{id}", updateShiftHoursHandler).Methods("PUT")
	r.HandleFunc("/api/shift-hours/{id}", deleteShiftHoursHandler).Methods("DELETE")
	r.HandleFunc("/api/event-titles", getEventTitlesHandler).Methods("GET")
	r.HandleFunc("/api/event-titles", createEventTitleHandler).Methods("POST")
	r.HandleFunc("/api/event-titles/{id}", updateEventTitleHandler).Methods("PUT")
	r.HandleFunc("/api/event-titles/{id}", deleteEventTitleHandler).Methods("DELETE")
	r.HandleFunc("/api/reports", getReportsHandler).Methods("GET")
	r.HandleFunc("/api/reports", createReportHandler).Methods("POST")
	r.HandleFunc("/api/reports/{id}", getReportHandler).Methods("GET")
	r.HandleFunc("/api/reports/{id}", updateReportHandler).Methods("PUT")
	r.HandleFunc("/api/reports/{id}", deleteReportHandler).Methods("DELETE")
	r.HandleFunc("/api/check-auth", checkAuthHandler).Methods("GET")

	fmt.Println("Server starting on :8084")
	log.Fatal(http.ListenAndServe(":8084", r))
}