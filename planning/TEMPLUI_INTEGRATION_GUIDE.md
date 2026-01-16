# templUI Integration Guide

This guide documents lessons learned from integrating [templUI](https://templui.io) into a Go web application using templ. It is intended to inform future agents or developers who may fully adopt this approach.

## Overview

**What is templUI?**
- A component library built on top of [templ](https://templ.guide/) (a Go templating engine)
- Provides pre-built, accessible UI components (buttons, cards, tables, badges, etc.)
- Uses Tailwind CSS for styling with CSS custom properties for theming
- Includes a CLI tool for adding components to your project

**Why consider templUI?**
- Type-safe templates that compile to Go code
- Component composition similar to React/Vue patterns
- IDE support with autocompletion for props
- Better refactoring support than text-based templates

## Installation

### Prerequisites

```bash
# Install templ CLI
go install github.com/a-h/templ/cmd/templ@latest

# Install templui CLI
go install github.com/templui/templui/cmd/templui@latest
```

### Initialize templUI in Your Project

Run from your project root:

```bash
templui init
```

This creates `.templui.json` with default paths. **Important:** Customize these paths before adding components.

### Configuration (.templui.json)

```json
{
  "componentsDir": "internal/templ/components",
  "utilsDir": "internal/templ/utils",
  "moduleName": "github.com/your-org/your-project",
  "jsDir": "web/static/js/templui",
  "jsPublicPath": "/static/js/templui"
}
```

**Key fields:**
- `componentsDir`: Where component `.templ` files will be placed
- `utilsDir`: Utilities like TwMerge (required by components)
- `moduleName`: Your Go module name (from go.mod)
- `jsDir`: Where JavaScript files for interactive components are placed
- `jsPublicPath`: URL path for serving those JS files

### Add Dependencies

```bash
go get github.com/a-h/templ
go get github.com/Oudwins/tailwind-merge-go
```

The `tailwind-merge-go` package is used by templUI's `TwMerge` utility to intelligently merge Tailwind classes without conflicts.

### Add Components

```bash
# Add individual components
templui add button
templui add card
templui add table
templui add badge

# Components are installed to your configured componentsDir
```

## Directory Structure

Recommended structure for a Go web application:

```
project-root/
├── .templui.json                    # templUI configuration
├── internal/
│   ├── templ/
│   │   ├── components/              # templUI library components
│   │   │   ├── button/
│   │   │   │   └── button.templ
│   │   │   ├── card/
│   │   │   │   └── card.templ
│   │   │   ├── table/
│   │   │   │   └── table.templ
│   │   │   └── badge/
│   │   │       └── badge.templ
│   │   ├── utils/
│   │   │   └── templui.go           # TwMerge and other utilities
│   │   └── admin/                   # Application-specific templ files
│   │       ├── layout/
│   │       │   └── base.templ       # Admin layout
│   │       └── dashboard/
│   │           ├── types.go         # Data structures
│   │           ├── page.templ       # Main page component
│   │           ├── stat_card.templ  # Custom component
│   │           └── orders_table.templ
│   └── handler/
│       └── admin/
│           └── dashboard_templ.go   # Handler that renders templ
└── web/
    └── static/
        ├── css/
        │   ├── input.css            # Tailwind input (imports theme)
        │   ├── templui-theme.css    # Theme CSS variables
        │   └── output.css           # Generated Tailwind CSS
        └── js/
            └── templui/             # templUI JavaScript files
```

## CSS Theme Setup

templUI components use CSS custom properties for theming. Create a theme file:

### web/static/css/templui-theme.css

```css
/* templUI Theme Configuration */
@custom-variant dark (&:where(.dark, .dark *));

@theme inline {
  --breakpoint-3xl: 1600px;
  --breakpoint-4xl: 2000px;
  --radius-sm: calc(var(--radius) - 4px);
  --radius-md: calc(var(--radius) - 2px);
  --radius-lg: var(--radius);
  --radius-xl: calc(var(--radius) + 4px);
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  /* ... additional chart and sidebar variables */
}

/* Light theme */
:root {
  --radius: 0.5rem;
  --background: oklch(0.985 0.002 90);
  --foreground: oklch(0.205 0.015 240);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.205 0.015 240);

  /* Primary color - customize for your brand */
  --primary: oklch(0.52 0.08 180);  /* Example: teal */
  --primary-foreground: oklch(0.985 0 0);

  /* Accent color */
  --accent: oklch(0.72 0.12 65);    /* Example: amber */
  --accent-foreground: oklch(0.205 0.015 240);

  /* ... complete theme variables */
}

/* Dark theme */
.dark {
  --background: oklch(0.145 0.01 240);
  --foreground: oklch(0.985 0 0);
  /* ... dark mode overrides */
}
```

### Import in Tailwind

```css
/* web/static/css/input.css */
@import "tailwindcss";
@import "./templui-theme.css";
@plugin "@tailwindcss/typography";

/* Your existing theme extensions */
@theme {
  --color-brand-primary: #2D7A7A;
  /* ... */
}
```

## Writing templ Components

### Basic Component Structure

```go
// internal/templ/admin/dashboard/stat_card.templ
package dashboard

import "github.com/your-org/project/internal/templ/utils"

type ColorVariant string

const (
    ColorDefault ColorVariant = "default"
    ColorAmber   ColorVariant = "amber"
    ColorGreen   ColorVariant = "green"
)

type StatCardProps struct {
    Label       string
    Value       string
    Description string
    Color       ColorVariant
}

// Method on props for computed classes
func (p StatCardProps) valueClasses() string {
    switch p.Color {
    case ColorAmber:
        return "text-amber-600 dark:text-amber-500"
    case ColorGreen:
        return "text-green-600 dark:text-green-500"
    default:
        return "text-zinc-950 dark:text-white"
    }
}

templ StatCard(props StatCardProps) {
    <div class="rounded-2xl bg-white p-6 ring-1 ring-zinc-950/5 dark:bg-zinc-900 dark:ring-white/10">
        <div class="text-sm/6 text-zinc-500 dark:text-zinc-400">
            { props.Label }
        </div>
        <div class={ utils.TwMerge("mt-2 text-3xl font-semibold", props.valueClasses()) }>
            { props.Value }
        </div>
        if props.Description != "" {
            <div class="mt-2 text-sm/6 text-zinc-500 dark:text-zinc-400">
                { props.Description }
            </div>
        }
    </div>
}
```

### Page Component with Layout

```go
// internal/templ/admin/dashboard/page.templ
package dashboard

import (
    "fmt"
    "github.com/your-org/project/internal/templ/admin/layout"
)

templ Page(data PageData) {
    @layout.AdminLayout("Dashboard", data.CurrentPath) {
        <div class="space-y-8">
            <!-- Stats Grid -->
            <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
                @StatCard(StatCardProps{
                    Label:       "Total Orders",
                    Value:       fmt.Sprintf("%d", data.OrderStats.TotalOrders),
                    Description: "Last 30 days",
                })
                @StatCard(StatCardProps{
                    Label: "Pending",
                    Value: fmt.Sprintf("%d", data.OrderStats.PendingOrders),
                    Color: ColorAmber,
                })
            </div>

            <!-- Use templUI components -->
            @OrdersTable(data.RecentOrders)
        </div>
    }
}
```

### Layout with Navigation

```go
// internal/templ/admin/layout/base.templ
package layout

import "strings"

type NavItem struct {
    Label  string
    Href   string
    Prefix string
}

var navItems = []NavItem{
    {Label: "Dashboard", Href: "/admin", Prefix: "/admin"},
    {Label: "Products", Href: "/admin/products", Prefix: "/admin/products"},
    // ...
}

func isActive(currentPath, itemHref, prefix string) bool {
    if itemHref == "/admin" {
        return currentPath == "/admin"
    }
    return strings.HasPrefix(currentPath, prefix)
}

templ AdminLayout(title, currentPath string) {
    <!DOCTYPE html>
    <html lang="en">
        <head>
            <meta charset="UTF-8"/>
            <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
            <title>{ title } - App Name</title>
            <link rel="stylesheet" href="/static/css/output.css"/>
            <script src="https://unpkg.com/htmx.org@1.9.10"></script>
            <script src="https://unpkg.com/alpinejs@3.13.3/dist/cdn.min.js" defer></script>
        </head>
        <body class="bg-zinc-50 dark:bg-zinc-950">
            @adminHeader(currentPath)
            <main class="mx-auto max-w-7xl">
                <div class="px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
                    { children... }
                </div>
            </main>
        </body>
    </html>
}

templ adminHeader(currentPath string) {
    <header class="border-b border-zinc-950/10 bg-white">
        <nav class="flex gap-6">
            for _, item := range navItems {
                <a
                    href={ templ.SafeURL(item.Href) }
                    class={ "px-3 py-2 text-sm font-medium",
                        templ.KV("text-zinc-950", isActive(currentPath, item.Href, item.Prefix)),
                        templ.KV("text-zinc-500 hover:text-zinc-950", !isActive(currentPath, item.Href, item.Prefix)) }
                >
                    { item.Label }
                </a>
            }
        </nav>
    </header>
}
```

## Data Types Pattern

Define data structures in a separate `.go` file alongside templ files:

```go
// internal/templ/admin/dashboard/types.go
package dashboard

import (
    "github.com/your-org/project/internal/domain"
    "github.com/jackc/pgx/v5/pgtype"
)

// PageData contains all data needed to render the dashboard
type PageData struct {
    CurrentPath  string
    OrderStats   OrderStats
    UserStats    UserStats
    RecentOrders []DisplayOrder
    Onboarding   *domain.OnboardingStatus
}

type OrderStats struct {
    TotalOrders         int64
    PendingOrders       int64
    TotalRevenueDollars string
}

// DisplayOrder represents an order formatted for display
// Pre-format data in the handler rather than in templates
type DisplayOrder struct {
    ID                 pgtype.UUID
    OrderNumber        string
    Status             string
    TotalDollars       string  // Pre-formatted: "123.45"
    CreatedAtFormatted string  // Pre-formatted: "Jan 2, 2006"
    CustomerName       string
}
```

## Handler Integration

```go
// internal/handler/admin/dashboard_templ.go
package admin

import (
    "fmt"
    "net/http"
    "time"

    "github.com/your-org/project/internal/handler"
    "github.com/your-org/project/internal/middleware"
    "github.com/your-org/project/internal/repository"
    "github.com/your-org/project/internal/templ/admin/dashboard"
    "github.com/jackc/pgx/v5/pgtype"
)

type DashboardTemplHandler struct {
    repo repository.Querier
}

func NewDashboardTemplHandler(repo repository.Querier) *DashboardTemplHandler {
    return &DashboardTemplHandler{repo: repo}
}

func (h *DashboardTemplHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Get tenant from middleware
    tenantUUID := middleware.GetTenantIDFromOperator(ctx)

    // Fetch data from database
    orderStats, err := h.repo.GetOrderStats(ctx, ...)
    if err != nil {
        handler.InternalErrorResponse(w, r, err)
        return
    }

    recentOrders, err := h.repo.ListOrders(ctx, ...)
    if err != nil {
        handler.InternalErrorResponse(w, r, err)
        return
    }

    // Transform database types to display types (pre-format in handler)
    displayOrders := make([]dashboard.DisplayOrder, len(recentOrders))
    for i, order := range recentOrders {
        displayOrders[i] = dashboard.DisplayOrder{
            ID:                 order.ID,
            OrderNumber:        order.OrderNumber,
            Status:             order.Status,
            TotalDollars:       fmt.Sprintf("%.2f", float64(order.TotalCents)/100),
            CreatedAtFormatted: order.CreatedAt.Time.Format("Jan 2, 2006"),
            CustomerName:       order.CustomerName,
        }
    }

    // Build page data
    pageData := dashboard.PageData{
        CurrentPath:  r.URL.Path,
        OrderStats:   dashboard.OrderStats{...},
        RecentOrders: displayOrders,
    }

    // Render templ component
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := dashboard.Page(pageData).Render(ctx, w); err != nil {
        handler.InternalErrorResponse(w, r, err)
        return
    }
}
```

## Build Process

### Generate templ Files

```bash
# Generate Go code from .templ files
templ generate

# Or watch mode for development
templ generate --watch
```

### Build CSS

```bash
# Build Tailwind CSS (ensure it scans .templ files)
./tailwind -i ./web/static/css/input.css -o ./web/static/css/output.css --minify
```

### Tailwind Configuration

Ensure your `tailwind.config.js` or content paths include `.templ` files:

```js
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./internal/templ/**/*.templ",  // Add this
    "./internal/templ/**/*.go",     // And this for Go helper funcs
  ],
  // ...
}
```

## Key Patterns and Lessons Learned

### 1. Props Pattern

templUI components use a Props struct pattern. Follow this for custom components:

```go
type Props struct {
    // Required fields first
    Title string

    // Optional fields with defaults
    Variant Variant
    Size    Size
    Class   string            // For additional classes
    Attributes templ.Attributes // For arbitrary HTML attributes
}

templ Component(props ...Props) {
    {{ var p Props }}
    if len(props) > 0 {
        {{ p = props[0] }}
    }
    // Set defaults if needed
    if p.Size == "" {
        {{ p.Size = SizeDefault }}
    }
    // Render...
}
```

### 2. TwMerge for Class Conflicts

Always use `utils.TwMerge()` when combining base classes with dynamic/override classes:

```go
templ Card(props Props) {
    <div class={ utils.TwMerge(
        "rounded-lg bg-white p-4",    // Base classes
        props.variantClasses(),        // Dynamic classes
        props.Class,                   // User overrides
    ) }>
        { children... }
    </div>
}
```

### 3. templ.KV for Conditional Classes

Use `templ.KV` for boolean class toggling:

```go
<a class={
    "px-3 py-2 text-sm",
    templ.KV("text-zinc-950 font-bold", isActive),
    templ.KV("text-zinc-500", !isActive),
}>
```

### 4. Pre-format Data in Handlers

Format dates, currency, etc. in the handler, not templates:

```go
// Handler
displayOrder.TotalDollars = fmt.Sprintf("%.2f", float64(order.TotalCents)/100)
displayOrder.CreatedAtFormatted = order.CreatedAt.Time.Format("Jan 2, 2006")

// Template (simple)
<td>{ order.TotalDollars }</td>
<td>{ order.CreatedAtFormatted }</td>
```

### 5. Alpine.js Integration

Alpine.js works seamlessly with templ:

```go
templ MobileMenu(currentPath string) {
    <div
        x-show="mobileMenuOpen"
        x-transition:enter="transition ease-out duration-200"
        x-transition:enter-start="opacity-0 -translate-y-1"
        x-transition:enter-end="opacity-100 translate-y-0"
        @click.away="mobileMenuOpen = false"
    >
        // Menu content
    </div>
}
```

### 6. HTMX Integration

templ works well with htmx patterns:

```go
templ DeleteButton(id string) {
    <button
        hx-delete={ fmt.Sprintf("/api/items/%s", id) }
        hx-confirm="Are you sure?"
        hx-target="closest tr"
        hx-swap="outerHTML"
        class="text-red-600 hover:text-red-800"
    >
        Delete
    </button>
}
```

### 7. Limited templUI Badge Variants

The templUI badge component only has: default, secondary, destructive, outline.
For status badges (pending, processing, shipped, etc.), create custom inline styles:

```go
templ StatusBadge(status string) {
    switch status {
    case "pending":
        <span class="inline-flex items-center rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-800">
            { status }
        </span>
    case "shipped":
        <span class="inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800">
            { status }
        </span>
    // ... other statuses
    }
}
```

### 8. templ.SafeURL for Dynamic URLs

```go
<a href={ templ.SafeURL(item.Href) }>{ item.Label }</a>
<a href={ templ.SafeURL(fmt.Sprintf("/admin/orders/%s", order.ID)) }>View</a>
```

## Gotchas and Troubleshooting

### 1. CSS Variables Must Be Complete

templUI components reference many CSS variables. If any are missing, components may appear broken. Use the full theme file from templUI docs as a starting point.

### 2. OKLCH Color Space

templUI uses OKLCH colors which provide better perceptual uniformity:
```css
--primary: oklch(0.52 0.08 180);  /* lightness chroma hue */
```

Convert your brand hex colors to OKLCH using tools like [oklch.com](https://oklch.com).

### 3. templ Generate Must Run Before Build

Add to your build process:
```makefile
build: templ-generate css:build go-build

templ-generate:
    @templ generate
```

### 4. IDE Support

Install the templ VS Code extension for syntax highlighting and autocompletion in `.templ` files.

### 5. Package Names in templ

The package declaration in `.templ` files must match the directory name:
```
internal/templ/admin/dashboard/page.templ
→ package dashboard
```

## Migration Strategy

For gradual migration from Go html/template:

1. Create parallel routes (`/admin` vs `/admin-templ`)
2. Convert one page at a time
3. Share handlers initially (same data fetching, different rendering)
4. Remove old templates once templ version is validated

Example route setup:
```go
// Keep existing route
r.Handle("/admin", existingDashboardHandler)

// Add templ version for comparison
r.Handle("/admin/dashboard-templ", templDashboardHandler)
```

## Hiri-Specific Implementation Notes

Based on the dashboard migration, here are specific patterns for converting Hiri admin pages:

### File Naming Convention

**CRITICAL:** Never name handler files with `_templ.go` suffix. The `templ generate` command deletes all `*_templ.go` files, assuming they are generated files.

```
✗ internal/handler/admin/dashboard_templ.go  <- Will be deleted!
✓ internal/handler/admin/dashboard_new.go    <- Safe
✓ internal/handler/admin/products_v2.go      <- Safe
```

### Directory Structure for Admin Pages

```
internal/templ/admin/
├── layout/
│   └── base.templ              # Shared admin layout (already created)
├── dashboard/                  # ✓ Converted
│   ├── types.go
│   ├── components.templ
│   └── page.templ
├── products/                   # Next candidate
│   ├── types.go                # ProductListData, ProductDetailData, DisplayProduct
│   ├── components.templ        # ProductCard, SKUTable, ImageGallery
│   ├── list.templ              # Products list page
│   └── detail.templ            # Product detail page
├── orders/
│   ├── types.go
│   ├── components.templ        # OrderRow, ShipmentCard, RefundHistory
│   ├── list.templ
│   └── detail.templ
└── customers/
    ├── types.go
    ├── components.templ
    ├── list.templ
    └── detail.templ
```

### Conversion Checklist for Each Page

1. **Create types.go** - Define typed structs for page data
   - Move away from `map[string]interface{}`
   - Pre-format dates, currency in handler (not template)
   - Convert UUIDs to strings for template use

2. **Create components.templ** - Extract reusable components
   - Status badges specific to that domain
   - Table rows with proper typing
   - Cards and sections

3. **Create page.templ** - Compose the full page
   - Import layout: `"github.com/dukerupert/hiri/internal/templ/admin/layout"`
   - Use `@layout.AdminLayout(title, currentPath) { ... }`

4. **Create handler** (e.g., `products_new.go`)
   - Same data fetching as existing handler
   - Transform to typed PageData
   - Render with `page.Page(data).Render(ctx, w)`

5. **Update routes**
   - Add to `AdminDeps` struct in `deps.go`
   - Register route in `admin.go`
   - Initialize in `main.go`

### Shared Components to Reuse

These components are already available in `internal/templ/admin/dashboard/`:

- `StatCard` - Metric display with color variants
- `PageHeader` - Title + description
- `Heading` - Semantic headings (h1-h3)
- `OrderStatusBadge` - Status badges (pending, processing, shipped, etc.)
- `EmptyState` - No data placeholder

Consider moving commonly-used components to a shared location:
```
internal/templ/admin/shared/
├── badges.templ      # All status badges
├── cards.templ       # StatCard, InfoCard, etc.
├── tables.templ      # Table wrapper, pagination
└── forms.templ       # Input, Select, Textarea wrappers
```

### Priority Order for Remaining Pages

Based on complexity and usage:

1. **Products list/detail** - Core functionality, good table/form practice
2. **Orders list/detail** - Similar patterns, status workflows
3. **Customers list/detail** - Wholesale approval UI
4. **Invoices** - Good HTMX integration practice
5. **Settings pages** - Forms and validation
6. **Reports** - Charts and data visualization

### HTMX Partial Responses

For HTMX partial updates, create separate templ components:

```go
// internal/templ/admin/orders/partials.templ
templ RefundHistoryPartial(refunds []Refund) {
    // Renders just the refund history section
    // No layout wrapper
}
```

Handler returns partial:
```go
if r.Header.Get("HX-Request") == "true" {
    return partials.RefundHistoryPartial(refunds).Render(ctx, w)
}
return page.OrderDetail(data).Render(ctx, w)
```

## Post-Migration Consolidation

After completing the templ migration for all admin pages, consolidate handlers to reduce code duplication.

### The Problem: Duplicate Handlers

During migration, you'll have pairs of handlers:

```go
// Old handler with html/template (mutations + old rendering)
type ProductHandler struct {
    repo     repository.Querier
    renderer *handler.Renderer  // old html/template renderer
    storage  storage.Storage
}

// New handler with templ (templ rendering only)
type ProductTemplHandler struct {
    repo    repository.Querier
    storage storage.Storage
}
```

This results in:
- Duplicate structs with similar dependencies
- Routes split between two handlers
- Maintenance overhead

### Consolidation Strategy

1. **Rename TemplHandler to Handler** - The templ handler becomes THE handler
2. **Merge mutation methods** - Copy POST/PUT/DELETE methods from old handler to new
3. **Remove renderer dependency** - templ doesn't need the old `*handler.Renderer`
4. **Update deps.go** - Remove duplicate handler fields
5. **Update routes** - Point all routes to consolidated handler
6. **Update main.go** - Simplify handler instantiation

### Step-by-Step Process

**1. Merge handler files**

```go
// Before: products_new.go
type ProductTemplHandler struct {
    repo    repository.Querier
    storage storage.Storage
}

func NewProductTemplHandler(repo repository.Querier, storage storage.Storage) *ProductTemplHandler {
    return &ProductTemplHandler{repo: repo, storage: storage}
}

func (h *ProductTemplHandler) List(w http.ResponseWriter, r *http.Request) { ... }
func (h *ProductTemplHandler) Detail(w http.ResponseWriter, r *http.Request) { ... }
func (h *ProductTemplHandler) ShowForm(w http.ResponseWriter, r *http.Request) { ... }

// After: products.go (merged)
type ProductHandler struct {
    repo    repository.Querier
    storage storage.Storage
}

func NewProductHandler(repo repository.Querier, storage storage.Storage) *ProductHandler {
    return &ProductHandler{repo: repo, storage: storage}
}

// Templ rendering methods
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) { ... }
func (h *ProductHandler) Detail(w http.ResponseWriter, r *http.Request) { ... }
func (h *ProductHandler) ShowForm(w http.ResponseWriter, r *http.Request) { ... }

// Mutation methods (copied from old handler)
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) { ... }
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) { ... }
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) { ... }
```

**2. Update deps.go**

```go
// Before
type AdminDeps struct {
    ProductHandler      *admin.ProductHandler       // old
    ProductTemplHandler *admin.ProductTemplHandler  // new
    // ... duplicates for each domain
}

// After
type AdminDeps struct {
    ProductHandler *admin.ProductHandler  // consolidated
    // ... single handler per domain
}
```

**3. Update routes**

```go
// Before
admin.Get("/admin/products", deps.ProductTemplHandler.List)
admin.Post("/admin/products", deps.ProductHandler.Create)  // different handler!

// After
admin.Get("/admin/products", deps.ProductHandler.List)
admin.Post("/admin/products", deps.ProductHandler.Create)  // same handler
```

**4. Update main.go**

```go
// Before
adminDeps := routes.AdminDeps{
    ProductHandler:      admin.NewProductHandler(repo, renderer, fileStorage),
    ProductTemplHandler: admin.NewProductTemplHandler(repo, fileStorage),
}

// After (no renderer needed)
adminDeps := routes.AdminDeps{
    ProductHandler: admin.NewProductHandler(repo, fileStorage),
}
```

### Helper Functions to Preserve

During consolidation, ensure helper functions are preserved:

```go
// These helpers may exist in old handlers - copy them to consolidated handler

// isCSVFile checks if filename ends with .csv
func isCSVFile(filename string) bool {
    return strings.HasSuffix(strings.ToLower(filename), ".csv")
}

// usStateCodes returns list of US state codes for tax rate forms
func usStateCodes() []string {
    return []string{"AL", "AK", "AZ", ...}
}

// getRefundReasons returns list of refund reason options
func getRefundReasons() []map[string]string { ... }
```

### HTMX Partials

For htmx partial responses, add types to the templ types.go file:

```go
// internal/templ/admin/orders/types.go

// RefundHistoryData contains data for the refund history partial
type RefundHistoryData struct {
    Refunds []DisplayRefund
    OrderID string
}
```

And create corresponding templ components:

```go
// internal/templ/admin/orders/refund.templ

// RefundHistory renders the refund history as an htmx partial
templ RefundHistory(data RefundHistoryData) {
    if len(data.Refunds) > 0 {
        <div class="divide-y divide-zinc-950/5 dark:divide-white/5">
            for _, refund := range data.Refunds {
                @refundHistoryItem(refund)
            }
        </div>
    } else {
        <p class="text-sm text-zinc-500 dark:text-zinc-400">No refunds yet.</p>
    }
}
```

### Consolidation Checklist

For each handler pair:

- [ ] Rename struct from `XxxTemplHandler` to `XxxHandler`
- [ ] Rename constructor from `NewXxxTemplHandler` to `NewXxxHandler`
- [ ] Update all method receivers from `(h *XxxTemplHandler)` to `(h *XxxHandler)`
- [ ] Copy mutation methods (Create, Update, Delete) from old handler
- [ ] Copy helper functions used by mutation methods
- [ ] Remove `renderer *handler.Renderer` from struct and constructor
- [ ] Delete old handler file
- [ ] Rename `xxx_new.go` to `xxx.go`
- [ ] Update deps.go to remove TemplHandler field
- [ ] Update routes to use consolidated handler
- [ ] Update main.go handler instantiation
- [ ] Build and test

### Results from Hiri Consolidation

After consolidating 16 handler pairs:

- **Lines deleted:** ~11,000 (old html/template files + duplicate handlers)
- **Lines added:** ~2,300 (helper functions, missing types)
- **Net reduction:** ~8,700 lines
- **Single source of truth:** Each domain has one handler

## Summary

**Benefits observed:**
- Type-safe templates catch errors at compile time
- IDE autocompletion for props
- Component composition feels natural
- Good integration with htmx and Alpine.js
- Post-migration consolidation reduces code by ~50%

**Considerations:**
- Additional build step (templ generate)
- Learning curve for templ syntax
- Some templUI components may need customization
- Theme setup requires complete CSS variable definitions
- **Never use `_templ.go` suffix for handler files**
- Plan for post-migration consolidation to avoid long-term handler duplication

**Recommended for:**
- New projects with significant UI complexity
- Teams comfortable with component-based architecture
- Projects already using Tailwind CSS