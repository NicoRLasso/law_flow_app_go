# Law Flow App - Go Fullstack

A modern fullstack web application built with Go, featuring a REST API and server-side rendered templates.

## 🚀 Tech Stack

- **[Echo](https://echo.labstack.com/)** - High performance, extensible web framework
- **[GORM](https://gorm.io/)** - The fantastic ORM library for Golang
- **[SQLite](https://www.sqlite.org/)** - Lightweight, file-based database
- **[Templ](https://templ.guide/)** - Type-safe HTML templating for Go

## 📁 Project Structure

```
law_flow_app_go/
├── cmd/
│   └── server/          # Application entry point
│       └── main.go
├── config/              # Configuration management
│   └── config.go
├── db/                  # Database setup and migrations
│   ├── database.go
│   └── app.db          # SQLite database file (generated)
├── handlers/            # HTTP request handlers
│   ├── home.go
│   └── user.go
├── models/              # Data models
│   └── user.go
├── templates/           # Templ templates
│   └── index.templ
├── static/              # Static assets
│   ├── css/
│   │   └── style.css
│   ├── js/
│   └── images/
├── go.mod
├── go.sum
└── README.md
```

## 🛠️ Setup & Installation

### Prerequisites

- Go 1.24+ installed
- Templ CLI installed

### Installation Steps

1. **Clone or navigate to the project directory**
   ```bash
   cd /Users/nicoroch/Documents/apps/law_flow_app_go
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Install Templ CLI** (if not already installed)
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```

4. **Generate Templ files**
   ```bash
   templ generate
   ```

5. **Create environment file** (optional)
   ```bash
   cp .env.example .env
   ```

## 🏃 Running the Application

### Development Mode

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

### Build for Production

```bash
go build -o bin/server cmd/server/main.go
./bin/server
```

## 📡 API Endpoints

### Web Routes
- `GET /` - Home page

### API Routes (REST)
- `GET /api/users` - Get all users
- `GET /api/users/:id` - Get user by ID
- `POST /api/users` - Create new user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user

### Example API Request

**Create a user:**
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "securepassword"
  }'
```

**Get all users:**
```bash
curl http://localhost:8080/api/users
```

## 🗄️ Database

The application uses SQLite with GORM for ORM functionality. The database file is automatically created at `db/app.db` when you first run the application.

### Migrations

Migrations run automatically on application startup. To add new models:

1. Create your model in `models/`
2. Add it to the `AutoMigrate` call in `cmd/server/main.go`

## 🎨 Frontend Development

### Templ Templates

Templates are written in Templ (`.templ` files). After making changes:

```bash
templ generate
```

This generates Go code from your templates.

### Static Assets

Place your CSS, JavaScript, and images in the `static/` directory. They're served at `/static/*`.

## 🔧 Configuration

Environment variables (optional):

- `SERVER_PORT` - Server port (default: 8080)
- `DB_PATH` - Database file path (default: db/app.db)
- `ENVIRONMENT` - Environment mode (default: development)

## 📝 Development Workflow

1. **Make changes to templates**: Edit `.templ` files in `templates/`
2. **Generate Templ code**: Run `templ generate`
3. **Update handlers**: Modify files in `handlers/`
4. **Add models**: Create new models in `models/`
5. **Run the app**: `go run cmd/server/main.go`

## 🧪 Next Steps

- Add authentication and authorization
- Implement more complex business logic
- Add unit and integration tests
- Set up CI/CD pipeline
- Add more API endpoints
- Enhance frontend with JavaScript interactivity

## 📚 Resources

- [Echo Documentation](https://echo.labstack.com/docs)
- [GORM Documentation](https://gorm.io/docs/)
- [Templ Documentation](https://templ.guide/)
- [Go Documentation](https://go.dev/doc/)

## 📄 License

This project is open source and available under the MIT License.
