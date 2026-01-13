# Features Guide

This guide details the core functionalities available in LawFlowApp and how they benefit your practice.

## 🔐 Authentication & Security

Security is the backbone of LawFlowApp. We ensure your sensitive legal data remains protected at every level.

- **Secure Login**: Session-based authentication using Bcrypt for password hashing.
- **MFA Ready**: Architecture designed to easily incorporate Multi-Factor Authentication.
- **Password Recovery**: A secure, token-based self-service workflow for users who forget their passwords.
- **Automatic Session Management**: Expired sessions and tokens are automatically cleaned up by background jobs.

## 🏢 Multi-tenancy (Firm Management)

LawFlowApp is a multi-tenant system, meaning multiple organizations (Firms) can use the platform while remaining completely isolated from each other.

- **Firm Isolation**: No data ever leaks between firms. A user in Firm A can never see data from Firm B.
- **Customizable Setup**: Each firm can configure its own basic details, timezone, and environment during the setup flow.
- **Admin Control**: Firm administrators have full control over their organization's settings and member list.

## 👥 User Roles & Permissions

Granular control over who can do what within your firm.

| Role | Description |
| :--- | :--- |
| **Admin** | Full access to firm settings, billing, and user management. |
| **Lawyer** | Can manage cases, clients, and legal documents. |
| **Staff** | Administrative support with access to schedules and client records. |
| **Client** | Restricted access to view their own case status and documents. |

## ⚡ Interactive User Interface

Powered by **HTMX** and **Alpine.js**, LawFlowApp provides a desktop-like experience in the browser.

- **Partial Updates**: Only the parts of the page that change are updated, meaning no jarring full-page refreshes.
- **Real-time Feedback**: Instant validation and UI updates as you interact with the software.
- **Mobile Responsive**: Access your firm's data from anywhere, on any device.

---
*For technical implementation details, refer to the [System Architecture](architecture.md).*
