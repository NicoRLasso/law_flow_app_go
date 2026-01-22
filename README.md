# ⚖️ lexlegalcloud - Go Fullstack

A modern, high-performance legal management application built with **Go (Echo)**, **Templ**, **HTMX**, and **Alpine.js**.

## 🚀 Quick Start

### Prerequisites
- **Go 1.24+**
- **Make**

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
- **Styling:** [Tailwind CSS](https://tailwindcss.com/)
- **Rich Text:** Custom Implementation (Built from scratch)
- **Infrastructure:** Docker

## ✨ Key Features
- **Multi-tenant Core:** Multi-firm support with dedicated firm setup flow.
- **Secure Auth:** Session-based authentication with Bcrypt and secure cookies.
- **User Management:** Full CRUD for users with Role-Based Access Control (Admin/Lawyer/Staff/Client).
- **Document Engine:** Create, edit (Custom Editor), and generate PDFs from templates with dynamic variable substitution.
- **Internationalization (i18n):** Full support for English and Spanish.
- **SEO Optimized:** Built-in sitemap, robots.txt, and semantic HTML structure.
- **Modern UI:** Responsive dashboard, interactive forms, and real-time partial updates via HTMX.

## 📂 Project Structure
- `cmd/server/`: Application entry point.
- `handlers/`: HTTP request handlers (thin layer).
- `models/`: GORM database models.
- `services/`: Core business logic (Auth, Email, PDFs, i18n).
- `templates/`: `.templ` views (Layouts, Pages, Partials).
- `static/`: Compiled CSS, JS, and assets.
- `db/`: Database configuration.

## 🔧 Development Workflow (Makefile)
| Command | Description |
| :--- | :--- |
| `make dev` | **Standard Dev:** Templ generate + Air live-reload. |
| `make generate` | Manual Templ code generation. |
| `make fmt` | Formats all Go and Templ files. |
| `make test` | Runs the test suite. |
| `make build` | Compiles an optimized binary in `bin/server`. |
| `make create-user` | Interactive CLI to create an initial admin user. |
| `make docker-run` | Run the application in a Docker container. |

## 📚 Resources
- [Project Documentation](docs/)
- [Official Makefile](Makefile) (Full command list)

---
*Built with speed and type-safety in mind.*
