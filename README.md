# ⚖️ LawFlow - Go Fullstack

A modern, high-performance legal management application built with **Go (Echo)**, **Templ**, **HTMX**, and **Alpine.js**.

## 🚀 Quick Start

### Prerequisites
- **Go 1.24+**
- **Make**
- **Bun** (for CSS builds)

### Installation & Run
```bash
make install-deps  # Install tools (templ, air) and modules
make dev           # Start with live-reload (recommended)
```
The app will be available at `http://localhost:8080`.

## 🛠 Tech Stack
- **Backend:** [Go](https://go.dev/) + [Echo](https://echo.labstack.com/) + [GORM](https://gorm.io/) (SQLite)
- **Views:** [Templ](https://templ.guide/) (Type-safe HTML)
- **Frontend Logic:** [HTMX](https://htmx.org/) (AJAX/Partials) + [Alpine.js](https://alpinejs.dev/) (Local state)
- **Styling:** [Tailwind CSS v4](https://tailwindcss.com/)
- **Search:** SQLite FTS5 (Full-Text Search)
- **Rich Text:** Custom Implementation (Built from scratch)
- **Infrastructure:** Docker

## ✨ Key Features
- **Judicial Tracking:** Automated integration with the Colombian Judicial Branch API to track case updates.
- **Appointment System:** Complete scheduling, availability, and calendar management for law firms.
- **Multi-tenant Core:** Multi-firm support with dedicated firm setup flow and subscription management.
- **Full-Text Search:** High-performance search across cases and documents powered by SQLite FTS5.
- **Secure Auth:** Session-based authentication with Bcrypt and secure cookies.
- **Document Engine:** Create, edit (Custom Editor), and generate PDFs from templates with dynamic variable substitution.
- **Audit System:** Comprehensive logging of all system actions for security and compliance.
- **Internationalization (i18n):** Full support for English and Spanish.
- **Modern UI:** Responsive dashboard, interactive forms, and real-time partial updates via HTMX.

## 📂 Project Structure
- `cmd/server/`: Application entry point.
- `handlers/`: HTTP request handlers (thin layer).
- `models/`: GORM database models.
- `services/`: Core business logic (Auth, Email, PDFs, Judicial API, Appointments, i18n).
- `templates/`: `.templ` views (Layouts, Pages, Partials).
- `static/`: Compiled CSS, JS, and assets.
- `db/`: Database configuration and SQLite files.

## 🧪 Unit Testing
The project uses Go's built-in testing tool with a focus on business logic and services.

```bash
make unit-test
```

**Key Details:**
- **Coverage:** Runs with `-cover` to show code coverage.
- **Exclusions:** Automatically excludes `templates`, `static`, `models`, `db`, and `config` dirs to focus on logic.
- **Advanced Usage:** Pass arguments using `ARGS`, e.g., `make unit-test ARGS="-run TestAuth"`.

## 🔒 Security
The project includes automated security scanning tools to ensure code quality and safety.

```bash
make security
```

This command runs:
- **[gosec](https://github.com/securego/gosec):** Static code analysis for security issues in the Go code.
- **[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck):** Checks for known vulnerabilities in your Go dependencies.

The tools will be automatically installed if they are missing from your system.

## 🔧 Development Workflow (Makefile)
| Command | Description |
| :--- | :--- |
| `make dev` | **Standard Dev:** Templ generate + Air live-reload + CSS watch. |
| `make security` | Run security scans (gosec + govulncheck). |
| `make unit-test` | Runs the test suite with coverage. |
| `make generate` | Manual Templ code generation. |
| `make fmt` | Formats all Go and Templ files. |
| `make build` | Compiles an optimized binary in `bin/server`. |
| `make create-user` | Interactive CLI to create an initial admin user. |
| `make docker-run` | Run the application in a Docker container. |

## 📚 Resources
- [Project Documentation](docs/)
- [Official Makefile](Makefile) (Full command list)

---
*Built with speed and type-safety in mind.*
