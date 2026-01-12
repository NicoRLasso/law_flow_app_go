# Law Flow App - Go Fullstack

A modern fullstack web application built with Go, featuring a REST API, server-side rendered templates, and interactive frontend with HTMX, Alpine.js, and Tailwind CSS.

## 🚀 Tech Stack

### Backend
- **[Echo](https://echo.labstack.com/)** - High performance, extensible web framework
- **[GORM](https://gorm.io/)** - The fantastic ORM library for Golang
- **[SQLite](https://www.sqlite.org/)** - Lightweight, file-based database
- **[Templ](https://templ.guide/)** - Type-safe HTML templating for Go

### Frontend
- **[HTMX](https://htmx.org/)** - Access AJAX, CSS Transitions, WebSockets directly in HTML
- **[Alpine.js](https://alpinejs.dev/)** - Lightweight JavaScript framework for reactive UI
- **[Tailwind CSS](https://tailwindcss.com/)** - Utility-first CSS framework

## 🛠️ Setup & Installation

### Prerequisites

- Go 1.24+ installed
- Make (for using Makefile commands)

### Installation Steps

1. **Navigate to the project directory**
   ```bash
   cd /Users/nicoroch/Documents/apps/law_flow_app_go
   ```

2. **Install dependencies and tools**
   ```bash
   make install-deps
   ```
   This installs:
   - Go module dependencies
   - Templ CLI
   - Air (live-reload tool)

3. **Create environment file** (optional)
   ```bash
   cp .env.example .env
   ```

## 🏃 Running the Application

### Development Mode (Live-Reload) - Recommended
```bash
make dev
```
This will:
- Generate Templ templates
- Start Air for live-reload
- Watch for changes in `.go` and `.templ` files
- Auto-rebuild and restart server on changes

### Standard Run
```bash
make run
```

### Build for Production
```bash
make build
./bin/server
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
make generate
```

Or use `make dev` for automatic regeneration on file changes.

### Template Organization

- **Layouts**: `templates/layouts/` - Base layouts with CDN links
- **Pages**: `templates/pages/` - Full page templates
- **Partials**: `templates/partials/` - HTMX partial templates (fragments)
- **Components**: `templates/components/` - Reusable components

### HTMX Pattern

For server interactions, use HTMX:
```html
<button 
    hx-get="/htmx/users" 
    hx-target="#user-list"
    hx-swap="innerHTML">
    Load Users
</button>
```

### Alpine.js Pattern

For client-side state (modals, dropdowns):
```html
<div x-data="{ open: false }">
    <button @click="open = true">Open Modal</button>
    <div x-show="open">Modal content</div>
</div>
```

### Static Assets

Place your CSS, JavaScript, and images in the `static/` directory. They're served at `/static/*`.

## 🔧 Configuration

Environment variables (optional):

- `SERVER_PORT` - Server port (default: 8080)
- `DB_PATH` - Database file path (default: db/app.db)
- `ENVIRONMENT` - Environment mode (default: development)

## 📝 Development Workflow

### Using Makefile

```bash
make help          # Show all available commands
make install-deps  # Install dependencies and tools
make generate      # Generate Templ templates
make dev           # Run with live-reload (recommended)
make run           # Run the application
make build         # Build production binary
make clean         # Clean build artifacts
make test          # Run tests
make fmt           # Format code (Go + Templ)
make tidy          # Tidy Go modules
```

### Live-Reload Development

1. **Start dev mode**: `make dev`
2. **Edit templates**: Modify `.templ` files in `templates/`
3. **Edit handlers**: Modify files in `handlers/`
4. **Auto-reload**: Air watches and rebuilds automatically
5. **Refresh browser**: See changes immediately

## 📚 Frontend Guidelines

See [`frontend_rules.md`](frontend_rules.md) for comprehensive guidelines on:
- When to use HTMX vs Alpine.js
- Template organization patterns
- HTMX request detection
- Best practices for Go + Templ + HTMX + Alpine.js

## 🧪 Testing

### Run Tests
```bash
make test
```

### Manual Testing

1. Start the server: `make dev`
2. Visit: `http://localhost:8080`
3. Test interactive features:
   - Click "Open Modal" to test Alpine.js
   - Click "Load Users" to test HTMX
   - Click "Toggle Dropdown" to test Alpine.js

## 🚀 Next Steps

- Add authentication and authorization
- Implement more CRUD interfaces with HTMX
- Add form validation with Alpine.js
- Create more partial templates for dynamic content
- Set up CI/CD pipeline
- Add unit and integration tests
- Deploy to production

## 📖 Resources

- [Echo Documentation](https://echo.labstack.com/docs)
- [GORM Documentation](https://gorm.io/docs/)
- [Templ Documentation](https://templ.guide/)
- [HTMX Documentation](https://htmx.org/docs/)
- [Alpine.js Documentation](https://alpinejs.dev/)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [Go Documentation](https://go.dev/doc/)

## 📄 License

This project is open source and available under the MIT License.
