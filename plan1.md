# Implementation Plan: Judicial Update Notifications

## Current State Analysis

The `judicial_updater.go` job currently:
1. Runs daily (every 24 hours after startup)
2. Checks cases with `filing_number` and `status = OPEN`
3. Creates `JudicialProcess` records and syncs `JudicialProcessAction` records
4. **Gap**: It inserts new actions but doesn't track/notify users about changes

---

## Phase 1: Create Notification Model

Create a new model `models/notification.go`:

```go
package models

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Notification types
const (
    NotificationTypeJudicialUpdate = "JUDICIAL_UPDATE"
    NotificationTypeCaseUpdate     = "CASE_UPDATE"
    NotificationTypeSystem         = "SYSTEM"
)

type Notification struct {
    ID        string         `gorm:"type:uuid;primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    // Targeting
    FirmID string  `gorm:"type:uuid;not null;index" json:"firm_id"`
    UserID *string `gorm:"type:uuid;index" json:"user_id,omitempty"` // null = all firm users

    // Context
    CaseID                   *string `gorm:"type:uuid" json:"case_id,omitempty"`
    JudicialProcessActionID  *string `gorm:"type:uuid" json:"judicial_process_action_id,omitempty"`

    // Content
    Type    string `gorm:"not null" json:"type"`
    Title   string `gorm:"not null" json:"title"`
    Message string `gorm:"type:text" json:"message"`
    LinkURL string `json:"link_url,omitempty"` // e.g., "/cases/{case_id}"

    // Read tracking
    ReadAt *time.Time `json:"read_at,omitempty"`

    // Relationships
    Firm   Firm  `gorm:"foreignKey:FirmID" json:"-"`
    User   *User `gorm:"foreignKey:UserID" json:"-"`
    Case   *Case `gorm:"foreignKey:CaseID" json:"case,omitempty"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
    if n.ID == "" {
        n.ID = uuid.New().String()
    }
    return nil
}

func (Notification) TableName() string {
    return "notifications"
}

func (n *Notification) IsRead() bool {
    return n.ReadAt != nil
}
```

---

## Phase 2: Modify Judicial Updater Job

Update `services/jobs/judicial_updater.go` to create notifications when new actions are found:

**Key changes:**
1. After successfully inserting a new `JudicialProcessAction`, create a `Notification`
2. Target the notification to:
   - The assigned lawyer (`AssignedToID` from Case)
   - Firm admins
   - Optionally the client

```go
// Inside the loop where new actions are created:
if exists == 0 {
    newAction := models.JudicialProcessAction{...}
    if err := database.Create(&newAction).Error; err != nil {
        log.Printf("[JOB] Failed to create action %s: %v", action.ExternalID, err)
    } else {
        // Create notification for new judicial update
        createJudicialUpdateNotification(database, c, newAction)
    }
}

func createJudicialUpdateNotification(db *gorm.DB, c models.Case, action models.JudicialProcessAction) {
    notification := models.Notification{
        FirmID:                  c.FirmID,
        CaseID:                  &c.ID,
        JudicialProcessActionID: &action.ID,
        Type:                    models.NotificationTypeJudicialUpdate,
        Title:                   fmt.Sprintf("Nueva actuación: %s", action.Type),
        Message:                 action.Annotation,
        LinkURL:                 fmt.Sprintf("/cases/%s", c.ID),
    }
    
    // Target assigned lawyer if exists
    if c.AssignedToID != nil {
        notification.UserID = c.AssignedToID
    }
    
    db.Create(&notification)
}
```

---

## Phase 3: Create Notification Handlers & Service

### 3.1. Create service `services/notification_service.go`:
- `GetUnreadNotifications(firmID, userID string) ([]Notification, error)`
- `MarkAsRead(notificationID, userID string) error`
- `MarkAllAsRead(firmID, userID string) error`
- `GetNotificationCount(firmID, userID string) (int64, error)`

### 3.2. Create handlers `handlers/notification.go`:
- `GET /api/notifications` - List unread notifications (HTMX partial)
- `PATCH /api/notifications/:id/read` - Mark single as read
- `PATCH /api/notifications/read-all` - Mark all as read

### 3.3. Add routes in `cmd/server/main.go`:

```go
// In protected routes group
protected.GET("/api/notifications", handlers.GetNotificationsHandler)
protected.PATCH("/api/notifications/:id/read", handlers.MarkNotificationReadHandler)
protected.PATCH("/api/notifications/read-all", handlers.MarkAllNotificationsReadHandler)
```

---

## Phase 4: Update Dashboard UI

### 4.1. Update `dashboard_view_model.go` - Add notifications to stats:

```go
package pages

import "law_flow_app_go/models"

type DashboardStats struct {
    ActiveCases          int64
    TotalClients         int64
    CompletedMonthly     int64
    PendingTasks         int64
    RecentCases          []models.Case
    UpcomingAppointments []models.Appointment
    Notifications        []models.Notification  // NEW
    UnreadCount          int64                   // NEW
}
```

### 4.2. Update `handlers/dashboard.go` - Fetch notifications:

```go
// Fetch unread notifications
var notifications []models.Notification
db.Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL", 
    firm.ID, user.ID).
    Order("created_at DESC").
    Limit(5).
    Find(&notifications)
stats.Notifications = notifications

db.Model(&models.Notification{}).
    Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL",
        firm.ID, user.ID).
    Count(&stats.UnreadCount)
```

### 4.3. Update `dashboard.templ` - Replace Firm Info with Notifications:

Move the notification section **ABOVE** the `<!-- Content Grid -->`:

```templ
<!-- Notifications Section - NEW (Above Content Grid) -->
if len(stats.Notifications) > 0 {
    <div class="mb-8" id="notifications-section">
        <div class="flex justify-between items-center mb-4">
            <h3 class="font-serif font-bold text-lg flex items-center gap-2">
                <svg class="w-5 h-5 text-primary" ...>bell icon</svg>
                { i18n.T(ctx, "dashboard.notifications.title") }
                <span class="badge badge-primary badge-sm">{ fmt.Sprintf("%d", stats.UnreadCount) }</span>
            </h3>
            <button 
                hx-patch="/api/notifications/read-all"
                hx-target="#notifications-section"
                hx-swap="outerHTML"
                class="btn btn-ghost btn-xs">
                { i18n.T(ctx, "dashboard.notifications.mark_all_read") }
            </button>
        </div>
        <div class="space-y-3">
            for _, notif := range stats.Notifications {
                <div class="card bg-base-100 shadow border-l-4 border-primary" id={ "notification-" + notif.ID }>
                    <div class="card-body p-4 flex flex-row justify-between items-start">
                        <div>
                            <p class="font-bold text-sm">{ notif.Title }</p>
                            <p class="text-xs opacity-70 line-clamp-2">{ notif.Message }</p>
                            <p class="text-xs opacity-50 mt-1">{ notif.CreatedAt.Format("02 Jan 2006, 15:04") }</p>
                        </div>
                        <div class="flex gap-2">
                            if notif.LinkURL != "" {
                                <a href={ templ.SafeURL(notif.LinkURL) } class="btn btn-primary btn-xs">Ver</a>
                            }
                            <button 
                                hx-patch={ "/api/notifications/" + notif.ID + "/read" }
                                hx-target={ "#notification-" + notif.ID }
                                hx-swap="outerHTML"
                                class="btn btn-ghost btn-xs">
                                ✓
                            </button>
                        </div>
                    </div>
                </div>
            }
        </div>
    </div>
}

<!-- Content Grid -->
<div class={ "grid gap-8 mb-12", templ.KV("lg:grid-cols-2", true) }>
    ...
</div>

<!-- REMOVE or MOVE Firm Information section to settings/profile -->
```

---

## Phase 5: Add i18n Translations

Add to your locale files:

### `services/i18n/locales/es.json`:
```json
{
  "dashboard.notifications.title": "Notificaciones",
  "dashboard.notifications.mark_all_read": "Marcar todo como leído",
  "dashboard.notifications.new_judicial_action": "Nueva actuación judicial",
  "dashboard.notifications.empty": "No hay notificaciones nuevas"
}
```

### `services/i18n/locales/en.json`:
```json
{
  "dashboard.notifications.title": "Notifications",
  "dashboard.notifications.mark_all_read": "Mark all as read",
  "dashboard.notifications.new_judicial_action": "New judicial action",
  "dashboard.notifications.empty": "No new notifications"
}
```

---

## Phase 6: Migration

Add the new model to the AutoMigrate call in `cmd/server/main.go`:

```go
db.AutoMigrate(..., &models.Notification{})
```

---

## Summary of Files to Create/Modify

| File | Action |
|------|--------|
| `models/notification.go` | **CREATE** |
| `services/notification_service.go` | **CREATE** |
| `handlers/notification.go` | **CREATE** |
| `services/jobs/judicial_updater.go` | **MODIFY** - Add notification creation |
| `templates/pages/dashboard.templ` | **MODIFY** - Add notifications UI |
| `templates/pages/dashboard_view_model.go` | **MODIFY** - Add notification fields |
| `handlers/dashboard.go` | **MODIFY** - Fetch notifications |
| `cmd/server/main.go` | **MODIFY** - Add routes + migration |
| `services/i18n/locales/es.json` | **MODIFY** - Add translations |
| `services/i18n/locales/en.json` | **MODIFY** - Add translations |

---

## Implementation Order

1. ✅ Phase 1: Create Notification Model
2. ✅ Phase 6: Add to AutoMigrate (do this early so table exists)
3. ✅ Phase 3: Create Service & Handlers
4. ✅ Phase 4: Update Dashboard (view model, handler, template)
5. ✅ Phase 5: Add i18n translations
6. ✅ Phase 2: Modify Judicial Updater (last, so notifications are created with working UI)