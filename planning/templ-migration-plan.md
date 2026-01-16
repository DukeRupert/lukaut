# Templ/TemplUI Migration Plan for Lukaut

## Executive Summary

This plan outlines the migration of Lukaut from Go HTML templates to templ/templUI components. The migration is approximately 15% complete, with auth pages and the dashboard already converted.

**Current Status:**
- 28 templ files created (auth pages, dashboard, layouts, shared components)
- 17 templUI components installed
- Theme CSS and utilities configured
- 26 page templates + 16 partials remaining

**Estimated Scope:** ~45 page/partial conversions across 5 phases

---

## Phase 0: Infrastructure Verification (Prerequisites)

Before continuing migration, verify the following are properly configured:

### Checklist
- [x] `.templui.json` configured with correct paths
- [x] `internal/templ/utils/templui.go` exists with TwMerge utility
- [x] `web/static/css/templui-theme.css` has complete CSS variable definitions
- [x] `web/static/css/input.css` imports templui-theme.css
- [x] Tailwind config scans `.templ` files
- [x] `internal/templ/layouts/app.templ` - Main app layout
- [x] `internal/templ/layouts/auth.templ` - Auth pages layout
- [x] `internal/templ/shared/flash.templ` - Flash message component

### Missing Components to Add
```bash
# Components likely needed for migration
templui add spinner      # For loading states
templui add skeleton     # For loading placeholders
templui add modal        # For confirmation dialogs
templui add tabs         # For tabbed interfaces
templui add radio        # For radio button groups
templui add switch       # For toggle switches
templui add fileupload   # For file upload (if available)
```

### Build Commands
```bash
# Add to development workflow
templ generate           # Compile .templ files
templ generate --watch   # Watch mode
sqlc generate            # After query changes
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/output.css --watch
```

---

## Phase 1: Settings Pages (Low Complexity)

**Goal:** Convert isolated settings pages to establish patterns for form handling.

### Directory Structure
```
internal/templ/pages/settings/
├── types.go          # ProfileData, PasswordData, BusinessData
├── profile.templ     # User profile form
├── password.templ    # Change password form
└── business.templ    # Business settings form
```

### Tasks

#### 1.1 Profile Settings
- [ ] Create `internal/templ/pages/settings/types.go`
  - `ProfilePageData` struct with user fields
  - Form validation error types
- [ ] Create `internal/templ/pages/settings/profile.templ`
  - Use `@layouts.App()` wrapper
  - Use templUI `input`, `button`, `form` components
  - Include CSRF token
  - Handle flash messages
- [ ] Update `internal/handler/settings.go`
  - Add `ShowProfileTempl()` method
  - Transform data to `ProfilePageData`
- [ ] Add route in router

#### 1.2 Password Settings
- [ ] Create `internal/templ/pages/settings/password.templ`
  - Current password + new password + confirm fields
  - Password strength validation (client-side)
  - CSRF token
- [ ] Update handler with `ShowPasswordTempl()`

#### 1.3 Business Settings
- [ ] Create `internal/templ/pages/settings/business.templ`
  - Company name, address, phone, email fields
  - Logo upload (if applicable)
  - Report header customization
- [ ] Update handler with `ShowBusinessTempl()`

### Validation Criteria
- All forms submit successfully
- Validation errors display correctly
- Flash messages appear after save
- CSRF protection works

---

## Phase 2: Clients & Sites CRUD (Moderate Complexity)

**Goal:** Convert standard CRUD pages with list/form patterns.

### Directory Structure
```
internal/templ/pages/clients/
├── types.go          # ClientListData, ClientFormData, DisplayClient
├── components.templ  # ClientRow, ClientCard
├── index.templ       # List page
├── new.templ         # Create form
├── edit.templ        # Edit form
└── show.templ        # Detail view

internal/templ/pages/sites/
├── types.go
├── components.templ  # SiteRow
├── index.templ
├── new.templ
└── edit.templ
```

### Tasks

#### 2.1 Clients List Page
- [ ] Create `types.go` with:
  ```go
  type ClientListPageData struct {
      CurrentPath string
      Clients     []DisplayClient
      Flash       *FlashMessage
  }

  type DisplayClient struct {
      ID        string
      Name      string
      Email     string
      Phone     string
      SiteCount int
      CreatedAt string // Pre-formatted
  }
  ```
- [ ] Create `index.templ` using templUI `table` component
- [ ] Create `components.templ` with `ClientRow` component
- [ ] Add htmx delete confirmation pattern

#### 2.2 Clients Forms (New/Edit)
- [ ] Create `new.templ` and `edit.templ`
- [ ] Share form component between new/edit (partial)
- [ ] Handle validation errors inline

#### 2.3 Clients Detail Page
- [ ] Create `show.templ` with client info + linked sites list
- [ ] Add edit/delete action buttons

#### 2.4 Sites Pages
- [ ] Create parallel structure for sites
- [ ] Include client selector dropdown in forms
- [ ] Link to parent client from detail view

### Handler Updates
- [ ] Add `*Templ()` methods to `client.go` and `site.go`
- [ ] Transform repository types to Display types
- [ ] Pre-format dates, addresses in handler

---

## Phase 3: Regulations Search (Moderate Complexity)

**Goal:** Convert search interface with full-text search results.

### Directory Structure
```
internal/templ/pages/regulations/
├── types.go
├── components.templ  # RegulationCard, SearchForm
└── index.templ
```

### Tasks
- [ ] Create types with search result display structures
- [ ] Create `index.templ` with:
  - Search input with htmx-powered results
  - Results list/cards
  - Pagination (if applicable)
- [ ] Preserve htmx search-as-you-type functionality
- [ ] Handle empty state

---

## Phase 4: Inspections (High Complexity)

**Goal:** Convert the most complex pages with careful attention to interactive features.

### Directory Structure
```
internal/templ/pages/inspections/
├── types.go            # All inspection-related types
├── components.templ    # Reusable inspection components
├── index.templ         # List page
├── new.templ           # Create form with file upload
├── edit.templ          # Edit form
├── show.templ          # Detail view (most complex)
├── review.templ        # Single violation review
├── review_queue.templ  # Queue-based review
└── partials/
    ├── image_gallery.templ
    ├── violations_list.templ
    ├── analysis_status.templ
    └── photo_upload.templ
```

### Tasks

#### 4.1 Inspections List
- [ ] Create `index.templ` with filtering
- [ ] Status badges (pending, analyzing, complete)
- [ ] Actions (view, edit, delete, generate report)
- [ ] htmx delete confirmation

#### 4.2 Inspection New/Edit
- [ ] Create `new.templ` with:
  - Client selector
  - Site selector (filtered by client)
  - Date picker
  - Description textarea
  - **Multi-file upload** (complex)
- [ ] Create `edit.templ` with existing images management
- [ ] Handle image preview and removal

#### 4.3 Inspection Show (Most Complex)
**This is the largest single template (~1,500 lines). Break into components:**

- [ ] Create `show.templ` orchestrating sub-components
- [ ] Create `partials/image_gallery.templ`:
  - Alpine.js state for selected image
  - Thumbnail grid
  - Main image display with zoom
  - Image metadata
- [ ] Create `partials/analysis_status.templ`:
  - htmx polling for status updates
  - Progress indicators
  - Status badges
- [ ] Create `partials/violations_list.templ`:
  - List of detected violations
  - Regulation references
  - Accept/reject controls
- [ ] Keyboard shortcut support (V key for violations)
- [ ] htmx partial swaps for analysis updates

#### 4.4 Violation Review Pages
- [ ] Create `review.templ` for single violation editing
- [ ] Create `review_queue.templ` with:
  - Queue navigation (prev/next)
  - Keyboard shortcuts
  - htmx queue advancement
  - Violation form with regulations

### Critical Features to Preserve
1. **Real-time polling** - Analysis status updates via htmx
2. **Keyboard navigation** - V key for violations, arrow keys in queue
3. **Image gallery** - Alpine.js controlled state
4. **File upload** - Progress tracking, multiple files
5. **htmx partials** - Status updates, form submissions

---

## Phase 5: Remaining Pages & Consolidation

### 5.1 Auth Edge Cases
- [ ] Convert `forgot_password_sent.html`
- [ ] Convert `reset_password_invalid.html`

### 5.2 Public Pages
- [ ] Convert `public/home.html` (landing page)
- [ ] Create public layout if different from app layout

### 5.3 Error Pages
- [ ] Create templ error pages (404, 500, etc.)
- [ ] Update error handler to use templ

### 5.4 Email Templates
- [ ] Evaluate: Keep HTML or convert to templ?
- [ ] Email templates may stay as HTML for compatibility

### 5.5 Handler Consolidation
After all pages are converted, consolidate handlers:

For each domain (clients, sites, inspections, etc.):
- [ ] Merge `*TemplHandler` methods into main handler
- [ ] Remove old HTML rendering methods
- [ ] Remove `*handler.Renderer` dependency
- [ ] Update routes to point to consolidated handler
- [ ] Delete old HTML templates
- [ ] Rename `*_new.go` to `*.go` if applicable

---

## Shared Components to Create

During migration, extract these reusable components:

```
internal/templ/shared/
├── flash.templ          # ✅ Already exists
├── toast.templ          # ✅ Already exists
├── page_header.templ    # Title + description + actions
├── empty_state.templ    # No data placeholder
├── confirm_dialog.templ # htmx delete confirmations
├── status_badge.templ   # Inspection/analysis status badges
├── pagination.templ     # List pagination controls
├── breadcrumbs.templ    # Navigation breadcrumbs
└── loading_spinner.templ # For async operations
```

---

## Testing Strategy

### Per-Page Testing
1. **Visual comparison** - Screenshot old vs new
2. **Form submission** - All forms work, validation displays
3. **htmx interactions** - Partials swap correctly
4. **Alpine.js state** - Dropdowns, modals, galleries work
5. **Keyboard shortcuts** - Still functional
6. **Mobile responsiveness** - Layout adapts correctly

### Integration Testing
1. **Full user flow** - Login → Create inspection → Upload photos → Review violations → Generate report
2. **Error handling** - Invalid inputs, server errors display correctly
3. **CSRF protection** - Forms reject without valid token

### Build Verification
```bash
# After each conversion
templ generate
go build ./...
go test ./...
```

---

## Migration Checklist Template

Use this checklist for each page conversion:

```markdown
### Page: [page_name]

**Files to create:**
- [ ] `types.go` - Define PageData structs
- [ ] `page.templ` - Main page component
- [ ] `components.templ` - Page-specific components (if needed)

**Handler updates:**
- [ ] Add `Show*Templ()` method
- [ ] Transform repository data to display types
- [ ] Pre-format dates, currency, etc.

**Route updates:**
- [ ] Add templ route (parallel or replacement)

**Testing:**
- [ ] Form submissions work
- [ ] Validation errors display
- [ ] htmx interactions work
- [ ] Flash messages appear
- [ ] Mobile layout correct

**Cleanup (after validation):**
- [ ] Remove old HTML template
- [ ] Remove old handler method
- [ ] Update route to use templ version only
```

---

## Timeline Estimate

| Phase | Pages | Complexity | Estimate |
|-------|-------|------------|----------|
| Phase 0 | - | Setup | Already done |
| Phase 1 | 3 | Low | Quick |
| Phase 2 | 8 | Moderate | Medium effort |
| Phase 3 | 1 | Moderate | Quick |
| Phase 4 | 6 | High | Significant effort |
| Phase 5 | 4+ | Mixed | Medium effort |
| Consolidation | - | Cleanup | Medium effort |

**Recommended approach:** Complete one phase fully before moving to the next. This allows learning and pattern refinement.

---

## Key Patterns Reference

### Form with CSRF
```go
templ ProfileForm(data ProfileFormData) {
    <form method="POST" action="/settings/profile">
        <input type="hidden" name="csrf_token" value={ data.CSRFToken }/>
        @input.Input(input.Props{
            Type: input.TypeEmail,
            Name: "email",
            Value: data.Email,
        })
        @button.Button(button.Props{Type: button.TypeSubmit}) {
            Save Changes
        }
    </form>
}
```

### htmx Delete with Confirmation
```go
templ DeleteButton(id string) {
    <button
        hx-delete={ fmt.Sprintf("/clients/%s", id) }
        hx-confirm="Are you sure you want to delete this client?"
        hx-target="closest tr"
        hx-swap="outerHTML swap:1s"
        class="text-red-600 hover:text-red-800"
    >
        Delete
    </button>
}
```

### Alpine.js Integration
```go
templ ImageGallery(images []DisplayImage) {
    <div x-data="{ selected: 0 }">
        <div class="main-image">
            for i, img := range images {
                <img
                    x-show={ fmt.Sprintf("selected === %d", i) }
                    src={ img.URL }
                    alt={ img.Alt }
                />
            }
        </div>
        <div class="thumbnails">
            for i, img := range images {
                <button @click={ fmt.Sprintf("selected = %d", i) }>
                    <img src={ img.ThumbnailURL } alt={ img.Alt }/>
                </button>
            }
        </div>
    </div>
}
```

### Handler Data Transformation
```go
func (h *ClientHandler) ShowListTempl(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    clients, err := h.repo.ListClients(ctx, userID)
    if err != nil {
        // handle error
    }

    // Transform to display types
    displayClients := make([]pages.DisplayClient, len(clients))
    for i, c := range clients {
        displayClients[i] = pages.DisplayClient{
            ID:        c.ID.String(),
            Name:      c.Name,
            Email:     c.Email,
            CreatedAt: c.CreatedAt.Time.Format("Jan 2, 2006"),
        }
    }

    data := pages.ClientListPageData{
        CurrentPath: r.URL.Path,
        Clients:     displayClients,
        Flash:       getFlash(r),
    }

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    pages.ClientList(data).Render(ctx, w)
}
```

---

## References

- [TEMPLUI_INTEGRATION_GUIDE.md](./TEMPLUI_INTEGRATION_GUIDE.md) - Detailed patterns and lessons learned
- [templ-migration.md](./templ-migration.md) - Progress tracker
- [templ documentation](https://templ.guide/)
- [templUI documentation](https://templui.io/)
