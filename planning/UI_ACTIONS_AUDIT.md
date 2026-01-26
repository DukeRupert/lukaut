# Lukaut UI Actions Audit

This document catalogs every page in the application and the actions a user can take on each one. Use it as a starting point to evaluate whether the UI needs refactoring or changes.

---

## Table of Contents

1. [Public Pages](#1-public-pages)
2. [Authentication Pages](#2-authentication-pages)
3. [Dashboard](#3-dashboard)
4. [Inspections](#4-inspections)
5. [Clients](#5-clients)
6. [Regulations](#6-regulations)
7. [Settings](#7-settings)
8. [Admin](#8-admin)
9. [Global UI Elements](#9-global-ui-elements)

---

## 1. Public Pages

### Home Page (`GET /`)

| Action | Element | Destination/Effect |
|---|---|---|
| Get started | Primary CTA button | `/register` |
| Sign in | Secondary link/button | `/login` |
| View Features section | Scroll | Reads feature cards (AI Analysis, Regulation Lookup, One-Click Reports) |
| Navigate to Terms | Footer link | `/terms` |
| Navigate to Privacy | Footer link | `/privacy` |
| Navigate to Contact | Footer link | `/contact` |

---

## 2. Authentication Pages

### Login (`GET /login`)

| Action | Element | Destination/Effect |
|---|---|---|
| Submit login form | Form (email + password) | `POST /login` — authenticates and redirects to `/dashboard` |
| Toggle "Remember me" | Checkbox | Extends session duration |
| Go to Forgot Password | Text link | `/forgot-password` |
| Go to Register | Footer link | `/register` |

### Register (`GET /register`)

| Action | Element | Destination/Effect |
|---|---|---|
| Submit registration form | Form (name, email, password, confirm, terms) | `POST /register` — creates account, auto-login, sends verification email |
| Accept terms of service | Required checkbox | Links to `/terms` and `/privacy` |
| Enter invite code | Text input (conditional) | Validates invite code during registration |
| Go to Login | Footer link | `/login` |

### Forgot Password (`GET /forgot-password`)

| Action | Element | Destination/Effect |
|---|---|---|
| Submit email for reset | Form (email) | `POST /forgot-password` — sends reset email, shows confirmation |
| Go back to Login | Footer link | `/login` |

### Reset Password (`GET /reset-password?token=...`)

| Action | Element | Destination/Effect |
|---|---|---|
| Submit new password | Form (new password, confirm) | `POST /reset-password` — updates password with token |
| Request new reset link | Link (shown on invalid/expired token) | `/forgot-password` |

### Verify Email (`GET /verify-email?token=...`)

| Action | Element | Destination/Effect |
|---|---|---|
| Go to Login (success) | Button | `/login` |
| Resend verification (failure) | Button | `/resend-verification` |

### Resend Verification (`GET /resend-verification`)

| Action | Element | Destination/Effect |
|---|---|---|
| Submit email for re-verification | Form (email) | `POST /resend-verification` — sends new verification email |
| Go to Login | Footer link | `/login` |

---

## 3. Dashboard

### Dashboard (`GET /dashboard`)

| Action | Element | Destination/Effect |
|---|---|---|
| View stats | Read-only stat cards | Displays: Total Inspections, Reports Generated, Violations Found, This Month |
| Create new inspection | "New Inspection" button | `/inspections/new` |
| View inspection detail | Click table row | `/inspections/{id}` |
| Complete business profile (onboarding) | Banner link (shown if no business profile) | `/settings/business` |

---

## 4. Inspections

### Inspections List (`GET /inspections`)

| Action | Element | Destination/Effect |
|---|---|---|
| Create new inspection | "New Inspection" button | `/inspections/new` |
| View inspection detail | Click table row | `/inspections/{id}` |
| Navigate pages | Pagination controls (htmx) | Loads next/previous page via `GET /inspections?page=N` |

### New Inspection (`GET /inspections/new`)

| Action | Element | Destination/Effect |
|---|---|---|
| Fill form fields | Inputs: title, address (line 1, line 2, city, state, zip), date, weather, temperature, notes | Populates inspection data |
| Select existing client | Client dropdown | Sets client_id on inspection |
| Quick-create client | "Add Client" button in dropdown | Opens inline modal with client form; creates client via `POST /clients/quick` and populates dropdown |
| Submit form | "Create Inspection" button | `POST /inspections` — creates and redirects to detail page |
| Cancel | "Cancel" button | `/inspections` |

### Edit Inspection (`GET /inspections/{id}/edit`)

| Action | Element | Destination/Effect |
|---|---|---|
| Edit form fields | Same fields as New Inspection | Modifies existing data |
| Submit changes | "Save" button | `PUT /inspections/{id}` — updates and redirects |
| Cancel | "Cancel" button | `/inspections/{id}` |

### Inspection Detail (`GET /inspections/{id}`)

| Action | Element | Destination/Effect |
|---|---|---|
| Edit inspection | "Edit" button | `/inspections/{id}/edit` |
| Back to list | "Back to List" button | `/inspections` |
| **Photos Section** | | |
| Upload photos | File input or drag-and-drop zone | `POST /inspections/{id}/images` — uploads and triggers analysis |
| View full-size photo | "View" button on thumbnail | Opens `/images/{id}/original` in new tab |
| Delete photo | "Delete" button on thumbnail | `DELETE /inspections/{id}/images/{imageId}` (with confirmation) |
| Monitor analysis | Auto-polling (htmx, every 3s) | `GET /inspections/{id}/images` — refreshes gallery during analysis |
| **Violations Section** | | |
| Go to review queue | "Review" button or keyboard `r`/`v` | `/inspections/{id}/review/queue` |
| View violations summary | Summary panel | Shows counts: total, pending, confirmed, rejected |
| **Reports Section** | | |
| Generate report | "Generate Report" button | `POST /inspections/{id}/reports` — enqueues report generation job |
| Download report | Download link | `GET /reports/{id}/download` — downloads PDF or DOCX |
| **Status** | | |
| Update inspection status | Status dropdown/button | `PUT /inspections/{id}/status` |
| **Keyboard Shortcuts** | | |
| Go to review queue | `r` or `v` key | `/inspections/{id}/review/queue` |
| Upload photos | `u` key | Focuses file input |

### Violation Review Queue (`GET /inspections/{id}/review/queue`)

| Action | Element | Destination/Effect |
|---|---|---|
| Accept violation | "Accept" button or `a` key | `PUT /inspections/{id}/review/queue/violations/{vid}/status` with status=confirmed |
| Reject violation | "Reject" button or `r` key | `PUT /inspections/{id}/review/queue/violations/{vid}/status` with status=rejected |
| Edit violation | "Edit" button or `e` key | Toggles inline edit form |
| Save edit | "Save" button in edit form | `PUT /violations/{id}` via htmx |
| Cancel edit | "Cancel" button | Hides edit form |
| Next violation | "Next" button, `j` key, or `→` key | Advances to next violation in queue |
| Previous violation | "Previous" button, `k` key, or `←` key | Goes back to previous violation |
| Exit review queue | `Esc` key | Returns to `/inspections/{id}` |
| View full-size image | Click image | Opens original in new tab |
| **Queue Completion Screen** | | |
| Return to inspection | Button | `/inspections/{id}` |
| Generate report | Button | `POST /inspections/{id}/reports` |

### Violation Card (partial, appears on Inspection Detail)

| Action | Element | Destination/Effect |
|---|---|---|
| Accept violation | "Accept" button (green) | `PUT /violations/{id}/status` with status=confirmed |
| Reject violation | "Reject" button (gray) | `PUT /violations/{id}/status` with status=rejected |
| Revert to pending | "Revert" button (yellow, shown on confirmed/rejected) | `PUT /violations/{id}/status` with status=pending |
| Edit violation | "Edit" button | Toggles inline edit form (Alpine.js) |
| Save edits | "Save" in edit form | `PUT /violations/{id}` via htmx |
| Delete violation | "Delete" button | `DELETE /violations/{id}` (with hx-confirm dialog) |
| Expand/collapse notes | Arrow toggle | Shows/hides inspector notes section |
| View linked regulations | Read-only list | Displays OSHA regulations with "Primary" badge |
| Batch update statuses | Batch action controls | `PUT /violations/batch/status` |

### Manual Violation Creation

| Action | Element | Destination/Effect |
|---|---|---|
| Create manual violation | Form/button on inspection detail | `POST /inspections/{id}/violations` |

---

## 5. Clients

### Clients List (`GET /clients`)

| Action | Element | Destination/Effect |
|---|---|---|
| Create new client | "New Client" button | `/clients/new` |
| View client detail | Click table row | `/clients/{id}` |
| Navigate pages | Pagination controls (htmx) | Loads next/previous page |

### New Client (`GET /clients/new`)

| Action | Element | Destination/Effect |
|---|---|---|
| Fill client form | Input fields (name, contact info, etc.) | Populates client data |
| Submit form | "Create Client" button | `POST /clients` — creates and redirects |
| Cancel | "Cancel" button | `/clients` |

### Client Detail (`GET /clients/{id}`)

| Action | Element | Destination/Effect |
|---|---|---|
| Edit client | "Edit" button | `/clients/{id}/edit` |
| Delete client | "Delete" button | `DELETE /clients/{id}` (with confirmation) |
| View associated inspections | Read-only list | Links to inspection detail pages |

### Edit Client (`GET /clients/{id}/edit`)

| Action | Element | Destination/Effect |
|---|---|---|
| Edit form fields | Same fields as New Client | Modifies existing data |
| Submit changes | "Save" button | `PUT /clients/{id}` — updates and redirects |
| Cancel | "Cancel" button | `/clients/{id}` |

---

## 6. Regulations

### Regulations Browser (`GET /regulations`)

| Action | Element | Destination/Effect |
|---|---|---|
| Search regulations | Search input (debounced 500ms) | `GET /regulations/search?q=...` via htmx — filters results |
| Filter by category | Category dropdown | `GET /regulations/search?category=...` via htmx |
| View regulation detail | Click regulation card | Opens modal via `GET /regulations/{id}` (htmx into Alpine.js modal) |
| Close detail modal | Click outside or close button | Dismisses modal (Alpine.js) |
| Navigate results pages | Pagination controls (htmx) | Loads next/previous page of results |

### Regulation-Violation Linking (on Inspection Detail)

| Action | Element | Destination/Effect |
|---|---|---|
| Add regulation to violation | Link/button | `POST /violations/{vid}/regulations/{rid}` |
| Remove regulation from violation | Remove button | `DELETE /violations/{vid}/regulations/{rid}` |

---

## 7. Settings

### Profile Settings (`GET /settings`)

| Action | Element | Destination/Effect |
|---|---|---|
| Update name | Text input | Part of profile form |
| View email (read-only) | Disabled input | Displays current email |
| Update phone | Phone input | Part of profile form |
| Save profile | "Save" button | `POST /settings/profile` |
| Switch to Password tab | Tab navigation | `/settings/password` |
| Switch to Business tab | Tab navigation | `/settings/business` |

### Password Settings (`GET /settings/password`)

| Action | Element | Destination/Effect |
|---|---|---|
| Change password | Form (current password, new password, confirm) | `POST /settings/password` — invalidates all other sessions |
| Switch to Profile tab | Tab navigation | `/settings` |
| Switch to Business tab | Tab navigation | `/settings/business` |

### Business Settings (`GET /settings/business`)

| Action | Element | Destination/Effect |
|---|---|---|
| Update business info | Form (business name, address, etc.) | `POST /settings/business` |
| Switch to Profile tab | Tab navigation | `/settings` |
| Switch to Password tab | Tab navigation | `/settings/password` |

---

## 8. Admin

### Admin Dashboard (`GET /admin` — assumed)

| Action | Element | Destination/Effect |
|---|---|---|
| View platform stats | Read-only stat cards | Total users, inspections, reports, AI costs |
| View users by AI cost | Table | Displays user info, subscription status, verification, inspection count, AI cost, join date |
| View monthly stats | Cost cards | This month's values for all metrics |

### Admin Users (`GET /admin/users` — assumed)

| Action | Element | Destination/Effect |
|---|---|---|
| Browse users | Table | Lists all platform users with details |

---

## 9. Global UI Elements

### App Layout Sidebar Navigation

| Action | Element | Destination/Effect |
|---|---|---|
| Go to Dashboard | Nav link + icon | `/dashboard` |
| Go to Inspections | Nav link + icon | `/inspections` |
| Go to Clients | Nav link + icon | `/clients` |
| Go to Regulations | Nav link + icon | `/regulations` |
| Go to Settings | Nav link + icon | `/settings` |
| Toggle mobile menu | Hamburger button | Opens/closes mobile sidebar overlay |

### Top Bar / User Menu

| Action | Element | Destination/Effect |
|---|---|---|
| Open user menu | Avatar/name dropdown | Shows dropdown with options |
| Go to Settings | Menu item | `/settings` |
| Sign out | Menu item | `POST /logout` — invalidates session, redirects to `/login` |

### Global Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `?` | Toggle keyboard shortcuts help modal |
| `c` | Create new client (`/clients/new`) |
| `i` | Create new inspection (`/inspections/new`) |
| `g` then `h` | Go to Dashboard |
| `g` then `i` | Go to Inspections |
| `g` then `c` | Go to Clients |
| `g` then `r` | Go to Regulations |

### Toast Notifications

| Trigger | Effect |
|---|---|
| Form submission success | Green toast with success message (auto-dismiss 5s) |
| Form submission error | Red toast with error message |
| Background job completion | Toast notification (e.g., report ready) |

### Flash Messages

| Type | Appearance |
|---|---|
| Error | Red banner, dismissible |
| Success | Green banner, dismissible |
| Warning | Amber banner, dismissible |
| Info | Blue banner, dismissible |

---

## Summary Statistics

| Category | Count |
|---|---|
| **Total pages** | ~25 distinct pages |
| **Total HTTP routes** | ~60 routes |
| **CRUD entities** | Inspections, Clients, Violations, Reports, Regulations, Users |
| **Forms** | ~15 distinct forms |
| **Keyboard shortcuts** | 13 shortcuts (7 global + 6 review queue) |
| **htmx interactions** | ~20 dynamic interactions (polling, search, inline edits, pagination) |
| **Alpine.js components** | ~10 stateful components (modals, dropdowns, drag-drop, toggles) |

---

## Notes for UI Review

- **Billing/subscription UI scaffolded (stubs)** — billing tab added to settings nav, stub handlers and routes registered for checkout, portal, cancel, reactivate, and webhook. Backend Stripe integration still TODO. See `internal/handler/billing.go` and `internal/handler/webhook.go` for implementation notes.
- **Email verification is not enforced** — `RequireEmailVerified` middleware exists but is not applied to any routes.
- **No user profile photo/avatar upload** — avatars appear to be initials-based only.
