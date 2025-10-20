# Go Event Web Application

A comprehensive web application for managing daily reports, shift schedules, and event tracking built with Go backend and modern HTML/CSS/JavaScript frontend.

## Table of Contents

- [Features](#features)
- [Technologies Used](#technologies-used)
- [Project Structure](#project-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [Database Setup](#database-setup)
- [Running the Application](#running-the-application)
- [API Endpoints](#api-endpoints)
- [Frontend Pages](#frontend-pages)
- [Admin Panel](#admin-panel)
- [Development](#development)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [License](#license)

## Features

- **User Authentication**: Secure login/logout system with role-based access control
- **Daily Report Management**: Create, view, edit, and delete daily reports
- **Shift Management**: Define and manage shift schedules
- **Event Tracking**: Track various events with detailed information
- **Health Checks**: Monitor system health parameters
- **RCA (Root Cause Analysis)**: Document root cause analysis for critical events
- **Admin Panel**: Comprehensive administration interface for managing users, shifts, and event titles
- **Responsive Design**: Mobile-friendly interface that works on all devices
- **Data Visualization**: Reports dashboard with statistics and insights

## Technologies Used

### Backend
- **Go (Golang)**: Main programming language
- **SQLite**: Database for data storage
- **Gorilla Mux**: HTTP request multiplexer
- **JWT**: Authentication and authorization

### Frontend
- **HTML5**: Markup language
- **CSS3/Tailwind CSS**: Styling and responsive design
- **JavaScript (ES6+)**: Client-side functionality
- **Font Awesome**: Icons
- **Fetch API**: AJAX requests

### Development Tools
- **Go Modules**: Dependency management
- **Air**: Live reloading for Go applications

## Project Structure

```
Go-event-web-22/
├── main.go                 # Main application entry point
├── database.html           # Database initialization and management
├── go.mod                  # Go module dependencies
├── go.sum                  # Go module checksums
├── README.md               # Project documentation
├── static/                 # Static assets directory
│   ├── admin.html          # Admin panel interface
│   ├── login.html          # User login page
│   ├── report-form.html    # Daily report creation/editing form
│   ├── reports-list.html   # Reports listing and viewing
│   ├── 200.png             # Company logo
│   └── ...                 # Other static assets
└── ...                     # Other project files
```

## Installation

1. **Prerequisites**:
   - Go 1.16 or higher
   - Git

2. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd Go-event-web-22
   ```

3. **Install dependencies**:
   ```bash
   go mod tidy
   ```

## Configuration

The application uses environment variables for configuration. Create a `.env` file in the project root:

```env
# Server configuration
PORT=8080
HOST=localhost

# Database configuration
DB_PATH=./data.db

# JWT configuration
JWT_SECRET=your-secret-key-here
JWT_EXPIRATION=24h
```

## Database Setup

The application uses SQLite for data storage. The database is automatically initialized when the application starts:

1. On first run, the application will create a SQLite database file
2. Tables for users, shifts, reports, events, and other entities will be created automatically
3. Default admin user will be created with username `admin` and password ``

## Running the Application

### Development Mode

```bash
# Run directly
go run main.go

# Or with live reload (if air is installed)
air
```

### Production Mode

```bash
# Build the application
go build -o go-event-web main.go

# Run the built binary
./go-event-web
```

The application will be available at `http://localhost:8080`

## API Endpoints

### Authentication
- `POST /api/login` - User login
- `POST /api/logout` - User logout
- `GET /api/check-auth` - Check authentication status

### Users
- `GET /api/users` - Get all users
- `POST /api/users` - Create new user
- `PUT /api/users/{id}` - Update user
- `DELETE /api/users/{id}` - Delete user

### Shift Hours
- `GET /api/shift-hours` - Get all shift hours
- `POST /api/shift-hours` - Create new shift
- `PUT /api/shift-hours/{id}` - Update shift
- `DELETE /api/shift-hours/{id}` - Delete shift

### Event Titles
- `GET /api/event-titles` - Get all event titles
- `POST /api/event-titles` - Create new event title
- `PUT /api/event-titles/{id}` - Update event title
- `DELETE /api/event-titles/{id}` - Delete event title

### Reports
- `GET /api/reports` - Get all reports
- `GET /api/reports/{id}` - Get specific report
- `POST /api/reports` - Create new report
- `PUT /api/reports/{id}` - Update report
- `DELETE /api/reports/{id}` - Delete report

## Frontend Pages

### Public Pages
- `/static/login.html` - User login page
- `/` - Home page (redirects to login if not authenticated)

### User Pages
- `/static/report-form.html` - Create/edit daily reports
- `/static/reports-list.html` - View all reports with filtering options

### Admin Pages
- `/static/admin.html` - Administrative panel for managing:
  - Users
  - Shift schedules
  - Event titles
  - System reports and statistics

## Admin Panel

The admin panel provides comprehensive management capabilities:

### Users Management
- Add, edit, and delete users
- Assign roles (admin/user)
- View user details

### Shifts Management
- Define shift schedules with start/end times
- Manage shift names and descriptions

### Event Titles Management
- Create and manage event categories
- Maintain consistent event naming

### Reports Overview
- View system statistics
- Monitor report submissions
- Access health check data

To access the admin panel:
1. Login with an account that has admin privileges
2. Click the "Admin" link in the navigation bar

## Development

### Code Structure

The main application is structured as follows:

- `main.go` - Contains all HTTP handlers and application logic
- `database.html` - Database initialization and helper functions
- Static files in `static/` directory handle the frontend

### Adding New Features

1. Create new API endpoints in `main.go`
2. Add corresponding frontend functionality in the appropriate HTML file
3. Update this README to document new features

### Database Schema

The application uses the following tables:
- `users` - User accounts and authentication
- `shift_hours` - Shift schedule definitions
- `event_titles` - Event category definitions
- `reports` - Daily reports
- `report_shift_managers` - Many-to-many relationship between reports and users
- `report_event_titles` - Many-to-many relationship between reports and event titles
- `report_events_part3` - Events requiring RCA
- `report_events_part4` - Events not requiring RCA

## Deployment

### Server Requirements
- Go runtime environment
- Write permissions for the application directory
- Port 8080 available (or configured port)

### Deployment Steps

1. Build the application:
   ```bash
   go build -o go-event-web main.go
   ```

2. Copy the binary and static files to the server

3. Set environment variables as needed

4. Run the application:
   ```bash
   ./go-event-web
   ```

### Reverse Proxy (Nginx)

Example Nginx configuration:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Note**: This application is designed for internal use and should be deployed in a secure environment. Always use strong passwords and keep the JWT secret secure in production environments.#   d a i l y - r e p o r t - g o 
 
 
