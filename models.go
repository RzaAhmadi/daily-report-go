package main

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type ShiftHours struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type EventTitle struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type DailyReport struct {
	ID                 int          `json:"id"`
	ReportDate         string       `json:"report_date"`
	ShiftHours         *ShiftHours  `json:"shift_hours"`
	ShiftManagers      []User       `json:"shift_managers"`
	EventTitles        []EventTitle `json:"event_titles"`
	HealthPowerSources bool         `json:"health_power_sources"`
	HealthHumidityTemp bool         `json:"health_humidity_temp"`
	HealthFireSystem   bool         `json:"health_fire_system"`
	EventsPart3        []EventPart3 `json:"events_part3"`
	EventsPart4        []EventPart4 `json:"events_part4"`
	CreatedBy          User         `json:"created_by"`
	CreatedAt          string       `json:"created_at"`
}

type EventPart3 struct {
	ID           int    `json:"id"`
	EventSummary string `json:"event_summary"`
	Trigger      string `json:"trigger_info"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	RCANumber    string `json:"rca_number"`
}

type EventPart4 struct {
	ID           int    `json:"id"`
	EventSummary string `json:"event_summary"`
	Trigger      string `json:"trigger_info"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}