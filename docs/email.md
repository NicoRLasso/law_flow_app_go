# Email Functionalities

## Overview

| Email Type | Recipient | Trigger |
|-----------|-----------|---------|
| Appointment Confirmation | Client | New appointment booked |
| Lawyer Appointment Notification | Lawyer | New appointment booked |
| Appointment Reminder | Client | Scheduled job (daily) |
| Appointment Cancellation | Client | Client cancels appointment |
| Password Reset | User | Forgot password submitted |
| Firm Setup Completion | Admin | Firm setup completed |
| Case Acceptance | Client | Case request accepted |
| Lawyer Assignment | Lawyer | Case assigned |
| Case Request Rejection | Client | Case request rejected |
| Collaborator Added | Lawyer | Added to case |

## Key Files

- **Service:** `services/email.go`
- **Templates:** `templates/emails/*.html`, `templates/emails/*.txt`
- **Jobs:** `services/jobs/reminders.go`

## Configuration

Environment variables in `config/config.go`:
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`
- `EMAIL_FROM`, `EMAIL_FROM_NAME`
- `ENVIRONMENT` (development mode logs instead of sending)
