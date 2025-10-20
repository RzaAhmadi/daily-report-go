-- Create database
CREATE DATABASE daily_reports;

-- Connect to the database
\c daily_reports;

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Shift hours management
CREATE TABLE shift_hours (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Event titles management
CREATE TABLE event_titles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Daily reports
CREATE TABLE daily_reports (
    id SERIAL PRIMARY KEY,
    report_date DATE NOT NULL,
    shift_hours_id INTEGER REFERENCES shift_hours(id),
    health_power_sources BOOLEAN DEFAULT FALSE,
    health_humidity_temp BOOLEAN DEFAULT FALSE,
    health_fire_system BOOLEAN DEFAULT FALSE,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Report shift managers (many-to-many)
CREATE TABLE report_shift_managers (
    report_id INTEGER REFERENCES daily_reports(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id),
    PRIMARY KEY (report_id, user_id)
);

-- Report event titles (many-to-many)
CREATE TABLE report_event_titles (
    report_id INTEGER REFERENCES daily_reports(id) ON DELETE CASCADE,
    event_title_id INTEGER REFERENCES event_titles(id),
    PRIMARY KEY (report_id, event_title_id)
);

-- Part 3 events (with RCA)
CREATE TABLE report_events_part3 (
    id SERIAL PRIMARY KEY,
    report_id INTEGER REFERENCES daily_reports(id) ON DELETE CASCADE,
    event_summary TEXT,
    trigger_info TEXT,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    rca_number VARCHAR(50)
);

-- Part 4 events (without RCA)
CREATE TABLE report_events_part4 (
    id SERIAL PRIMARY KEY,
    report_id INTEGER REFERENCES daily_reports(id) ON DELETE CASCADE,
    event_summary TEXT,
    trigger_info TEXT,
    start_time TIMESTAMP,
    end_time TIMESTAMP
);

-- Insert default admin user (password: admin123)
INSERT INTO users (username, password, full_name, role) 
VALUES ('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'System Administrator', 'admin');

-- Insert sample data
INSERT INTO shift_hours (name, start_time, end_time) VALUES 
('Morning Shift', '06:00:00', '14:00:00'),
('Evening Shift', '14:00:00', '22:00:00'),
('Night Shift', '22:00:00', '06:00:00');

INSERT INTO event_titles (title) VALUES 
('System Maintenance'),
('Security Check'),
('Equipment Inspection'),
('Emergency Response'),
('Routine Monitoring');

INSERT INTO users (username, password, full_name, role) VALUES 
('manager1', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'John Manager', 'user'),
('manager2', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Jane Supervisor', 'user');