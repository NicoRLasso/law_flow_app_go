# Plan: Intelligent Document Automation (Automatización Documental Inteligente)

## Overview
Implement a document template system that auto-fills case/client data and generates professional PDFs for legal documents.

**User Choices:**
- Editor: TipTap (headless, customizable)
- PDF Engine: Chromedp (headless Chrome)
- Scope: Firm-level templates only
- Variables: Simple `{{variable}}` replacement

---

## Phase 1: Template System (Structure)

### 1.1 New Database Models

**Create `models/document_template.go`:**
```go
type DocumentTemplate struct {
    ID              string  // UUID
    FirmID          string  // Multi-tenant scoping
    Name            string  // e.g., "Demanda de Alimentos"
    Description     *string
    CategoryID      *string // FK to TemplateCategory
    Content         string  // HTML with {{variable}} placeholders
    Version         int     // For tracking changes
    IsActive        bool
    CreatedByID     string
    // PDF Settings
    PageOrientation string  // portrait, landscape
    PageSize        string  // letter, legal, A4
    MarginTop       int     // points (72 = 1 inch)
    MarginBottom    int
    MarginLeft      int
    MarginRight     int
}
```

**Create `models/template_category.go`:**
```go
type TemplateCategory struct {
    ID          string
    FirmID      string
    Name        string  // e.g., "Demandas", "Contratos"
    Description *string
    SortOrder   int
    IsActive    bool
}
```

**Create `models/generated_document.go`:**
```go
type GeneratedDocument struct {
    ID              string
    FirmID          string
    TemplateID      string
    TemplateVersion int
    CaseID          string
    Name            string
    FinalContent    string  // Rendered HTML snapshot
    FileName        string
    FilePath        string
    FileSize        int64
    GeneratedByID   string
    CaseDocumentID  *string // Link to auto-archived CaseDocument
}
```

### 1.2 Variable Dictionary

**Create `services/template_variables.go`:**

| Category | Variables |
|----------|-----------|
| Client | `{{client.name}}`, `{{client.email}}`, `{{client.phone}}`, `{{client.document_type}}`, `{{client.document_number}}`, `{{client.address}}` |
| Case | `{{case.number}}`, `{{case.title}}`, `{{case.description}}`, `{{case.status}}`, `{{case.domain}}`, `{{case.branch}}`, `{{case.subtypes}}`, `{{case.opened_at}}` |
| Firm | `{{firm.name}}`, `{{firm.address}}`, `{{firm.city}}`, `{{firm.phone}}`, `{{firm.billing_email}}`, `{{firm.info_email}}` |
| Lawyer | `{{lawyer.name}}`, `{{lawyer.email}}`, `{{lawyer.phone}}` |
| Dates | `{{today.date}}`, `{{today.date_long}}`, `{{today.year}}` |

### 1.3 API Endpoints

```
# Template Management (Admin only)
POST   /api/admin/templates              - Create template
PUT    /api/admin/templates/:id          - Update template
DELETE /api/admin/templates/:id          - Delete template
POST   /api/admin/templates/categories   - Create category
PUT    /api/admin/templates/categories/:id
DELETE /api/admin/templates/categories/:id

# Template Usage (Admin + Lawyer)
GET    /api/templates                    - List templates
GET    /api/templates/:id                - Get template
GET    /api/templates/variables          - Get variable dictionary
GET    /api/templates/categories         - Get categories

# Page
GET    /templates                        - Templates management page
```

### 1.4 Files to Create - Phase 1

| File | Purpose |
|------|---------|
| `models/document_template.go` | DocumentTemplate model |
| `models/template_category.go` | TemplateCategory model |
| `models/generated_document.go` | GeneratedDocument model |
| `services/template_variables.go` | Variable definitions |
| `handlers/template.go` | Template CRUD handlers |
| `templates/pages/templates.templ` | Templates list page |
| `templates/partials/template_table.templ` | Templates table |
| `templates/partials/template_form_modal.templ` | Create/Edit modal with TipTap editor |

---

## Phase 2: Substitution Engine (Logic)

### 2.1 Template Rendering Service

**Create `services/template_engine.go`:**
- `RenderTemplate(template, data) string` - Replace `{{variables}}` with real values
- `BuildTemplateDataFromCase(case, firm) TemplateData` - Extract all data from case relationships
- Simple regex-based replacement: `/\{\{([a-zA-Z0-9_.]+)\}\}/`

### 2.2 TipTap Editor Integration

**CDN dependencies to add in base layout:**
```html
<script src="https://cdn.jsdelivr.net/npm/@tiptap/core@2"></script>
<script src="https://cdn.jsdelivr.net/npm/@tiptap/starter-kit@2"></script>
```

**Editor features:**
- Variables sidebar (click to insert `{{variable}}`)
- Basic formatting (bold, italic, headings, lists)
- HTML output mode
- Live preview toggle

### 2.3 Preview Handler

```
GET /api/cases/:id/generate/preview?template_id=xxx
```
- Fetches case with all relationships (Client, Lawyer, Domain, Branch, Subtypes)
- Renders template with real data
- Returns HTML for preview display

### 2.4 Files to Create - Phase 2

| File | Purpose |
|------|---------|
| `services/template_engine.go` | Variable substitution logic |
| `handlers/template_preview.go` | Preview with case data |
| `templates/partials/template_editor.templ` | TipTap editor component |
| `templates/partials/variables_sidebar.templ` | Clickable variables list |

---

## Phase 3: PDF Generation (Export)

### 3.1 Chromedp PDF Generation

**Add dependency:**
```
go get github.com/chromedp/chromedp
```

**Create `services/pdf_generator.go`:**
```go
func GeneratePDF(htmlContent string, options PDFOptions) ([]byte, error) {
    // 1. Wrap HTML with legal document styles (Times New Roman, margins, etc.)
    // 2. Use chromedp to render HTML to PDF
    // 3. Return PDF bytes
}
```

**Legal document styles:**
- Font: Times New Roman, 12pt
- Line height: 1.5
- Margins: 1 inch (72pt) default
- Page numbering
- Justified text alignment

### 3.2 Document Generation Flow

```
POST /api/cases/:id/generate
```

1. Fetch template and case with relationships
2. Render template with case data
3. Allow optional manual edits (from form)
4. Generate PDF via chromedp
5. Save PDF to `uploads/firms/{firm_id}/cases/{case_id}/generated/`
6. Create `GeneratedDocument` record
7. Auto-create `CaseDocument` record (archive)
8. Return success with download link

### 3.3 Case Detail Integration

**Modify `templates/pages/case_detail.templ`:**
- Add "Generate Document" tab
- Template selector dropdown
- Preview panel
- Generate button
- Generated documents history

### 3.4 API Endpoints

```
GET  /api/cases/:id/generate              - Get generation UI partial
POST /api/cases/:id/generate              - Generate document
GET  /api/cases/:id/generate/preview      - Preview with case data
GET  /api/cases/:id/generated             - List generated documents
```

### 3.5 Files to Create - Phase 3

| File | Purpose |
|------|---------|
| `services/pdf_generator.go` | Chromedp PDF generation |
| `handlers/document_generation.go` | Generation handlers |
| `templates/partials/generate_document_tab.templ` | Generation UI |
| `templates/partials/document_finalize_modal.templ` | Final edit before PDF |
| `templates/partials/generated_documents_table.templ` | History of generated docs |

---

## Files to Modify

| File | Changes |
|------|---------|
| `cmd/server/main.go` | Add routes, register new models for AutoMigrate |
| `templates/pages/case_detail.templ` | Add "Generate Document" tab |
| `templates/components/navbar.templ` | Add "Templates" nav link (admin only) |
| `services/i18n/en.json` | Add template translations |
| `services/i18n/es.json` | Add Spanish translations |

---

## i18n Keys to Add

```json
{
  "templates": {
    "title": "Document Templates",
    "create": "Create Template",
    "edit": "Edit Template",
    "name": "Template Name",
    "description": "Description",
    "category": "Category",
    "content": "Content",
    "preview": "Preview",
    "variables": "Variables",
    "insert_variable": "Click to insert",
    "page_settings": "Page Settings",
    "generate": "Generate Document",
    "generated_success": "Document generated successfully",
    "select_template": "Select a template",
    "no_templates": "No templates available"
  },
  "case": {
    "detail": {
      "tab": {
        "generate": "Generate Document"
      }
    }
  }
}
```

---

## Implementation Order

### Step 1: Database Models & Migrations
1. Create `models/document_template.go`
2. Create `models/template_category.go`
3. Create `models/generated_document.go`
4. Add to AutoMigrate in `main.go`

### Step 2: Variable Dictionary
1. Create `services/template_variables.go`
2. Define all variable categories and mappings

### Step 3: Template CRUD
1. Create `handlers/template.go`
2. Create `templates/pages/templates.templ`
3. Create `templates/partials/template_table.templ`
4. Create `templates/partials/template_form_modal.templ`
5. Add routes to `main.go`
6. Add navbar link

### Step 4: TipTap Editor
1. Add TipTap CDN to base layout
2. Create `templates/partials/template_editor.templ`
3. Create `templates/partials/variables_sidebar.templ`

### Step 5: Template Engine
1. Create `services/template_engine.go`
2. Implement `RenderTemplate()` and `BuildTemplateDataFromCase()`

### Step 6: Preview Functionality
1. Create `handlers/template_preview.go`
2. Add preview route

### Step 7: PDF Generation
1. Add chromedp dependency
2. Create `services/pdf_generator.go`
3. Test PDF generation locally

### Step 8: Document Generation Flow
1. Create `handlers/document_generation.go`
2. Implement auto-archive to CaseDocument
3. Add generation routes

### Step 9: Case Detail Integration
1. Modify `templates/pages/case_detail.templ` to add tab
2. Create `templates/partials/generate_document_tab.templ`
3. Create `templates/partials/generated_documents_table.templ`

### Step 10: i18n & Polish
1. Add all translations to `en.json` and `es.json`
2. Test full flow
3. Handle edge cases (missing data, empty fields)

---

## Verification Plan

1. **Template CRUD**: Create a template, edit it, delete it
2. **Variable Insertion**: Insert variables in editor, verify they appear in content
3. **Preview**: Select a case, preview template with substituted data
4. **PDF Generation**: Generate PDF, verify formatting and content
5. **Auto-Archive**: Verify generated PDF appears in case documents
6. **Multi-tenant**: Verify templates are firm-scoped (can't see other firm's templates)

---

## Critical Reference Files

- `models/case.go` - Case model with relationships (lines 1-150)
- `models/case_document.go` - Pattern for document storage
- `handlers/case_log.go` - Pattern for CRUD handlers
- `templates/pages/case_detail.templ` - Page to modify for generation tab
- `cmd/server/main.go` - Router configuration (lines 100-300)
- `services/upload.go` - File storage patterns
