# Habeas Data Compliance - Law 1581 of 2012

## Overview

LexLegal Cloud implements comprehensive compliance with **Colombian Law 1581 of 2012** (Habeas Data) and its regulatory decree **1377 of 2013**. This document describes how the platform fulfills all legal requirements.

---

## Legal Requirements vs Implementation

### 1. Authorization (Art. 9)

**Requirement:** Prior, express, and informed authorization from the data subject.

**Implementation:**
- **Capture Mechanism:** `ConsentModal` component triggers automatically on login if no valid consent exists.
- **Explicit Action:** User must click "I Accept" after viewing the policy text (Law 1581).
- **Storage:** `ConsentLog` model stores immutable consent records with `Granted=true`.
- **Audit Trail:** Records IP address, User Agent, Timestamp, and Policy Version.
- **Revocation:** Users can revoke consent at any time (triggers mechanism to stop processing).
- `services/consent_service.go` handles all consent logic.

---

### 2. ARCO Rights (Art. 15-16)

| Right | Model Field | Handler |
|-------|-------------|---------|
| **Access** | `SubjectRightsRequest.Type="ACCESS"` | `GetComplianceARCORequestsHandler` |
| **Rectification** | `SubjectRightsRequest.Type="RECTIFY"` | `ResolveComplianceARCORequestHandler` |
| **Cancellation** | `SubjectRightsRequest.Type="CANCEL"` | `ExportComplianceUserDataHandler` |
| **Opposition** | `SubjectRightsRequest.Type="OPPOSITION"` | — |
| **Portability** | ZIP export at `/compliance/export` | `ExportComplianceUserDataHandler` |

---

### 3. Security Measures (Art. 17)

**Technical Measures:**
- AES-256-GCM encryption (`services/encryption_service.go`)
- Password hashing with bcrypt (12 rounds)
- CSP headers, rate limiting, CSRF protection
- SQLite WAL mode with encrypted storage

**Administrative Measures:**
- Role-based access control (RBAC)
- Immutable audit logs (`models/audit_log.go`)
- Session management with secure cookies

---

### 4. Data Breach Notification (Art. 18)

**Implementation:**
- `SecurityEventMonitor` detects suspicious activity
- `BuildBreachNotificationEmail` sends alerts
- 72-hour notification requirement tracked via timestamps
- Alert triggers: massive downloads, failed logins

---

### 5. Data Retention & Deletion (Art. 15, 21)

**Right to Delete:**
- **User Action:** Dedicated "Privacy & Data" tab in User Settings allows request submission.
- **Process:** Users select "CANCEL (Delete)" request type in ARCO form.
- **Outcome:** Admins review request -> If approved, data is anonymized or soft-deleted.
- **Proof:** `BuildDataDeletionCertificateEmail` generates proof of deletion.

**Retention Policy:**
- Inactive accounts: Retained for 5 years (audit/legal reasons).
- Soft delete: `DeletedAt` field used via GORM.
- Hard delete: Only upon specific legal requirement or expiration of retention period.

---

## Key Files

| Purpose | File |
|---------|------|
| Consent tracking | `models/consent_log.go` |
| ARCO requests | `models/subject_rights_request.go` |
| Encryption | `services/encryption_service.go` |
| Breach alerts | `services/security_monitor.go` |
| Compliance emails | `services/compliance_emails.go` |
| UI Dashboard | `/compliance` route |

---

## Supervisor Contact

Per Art. 23, data subjects may contact the **Superintendencia de Industria y Comercio (SIC)** for complaints.
