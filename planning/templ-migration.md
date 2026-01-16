# Templ Migration Tracker

Track the progress of converting Go HTML templates to templ components.

## Completed

### Auth Pages
- [x] `auth/login` - Login form with CSRF
- [x] `auth/register` - Registration form with CSRF
- [x] `auth/forgot_password` - Forgot password form with CSRF
- [x] `auth/reset_password` - Reset password form with CSRF
- [x] `auth/verify_email` - Email verification result page
- [x] `auth/resend_verification` - Resend verification form with CSRF

### Settings Pages
- [x] `settings/profile` - User profile form
- [x] `settings/password` - Change password form
- [x] `settings/business` - Business profile settings

### Layouts
- [x] `layouts/app` - Main app layout with sidebar
- [x] `layouts/auth` - Auth pages layout

### Clients CRUD
- [x] `clients/index` - List clients with pagination
- [x] `clients/new` - Create client form
- [x] `clients/edit` - Edit client form
- [x] `clients/show` - Client detail view

### Sites CRUD
- [x] `sites/index` - List sites
- [x] `sites/new` - Create site form with client selection
- [x] `sites/edit` - Edit site form

### Regulations
- [x] `regulations/index` - Regulation search/list with htmx search and Alpine.js modal
- [x] `regulations/search` - htmx search results partial
- [x] `regulations/detail` - Regulation detail modal partial

## In Progress

### Dashboard
- [ ] `dashboard` - Main dashboard after login

## Not Started

### Public Pages
- [ ] `public/home` - Marketing landing page

### Inspections (Complex)
- [ ] `inspections/index` - List inspections
- [ ] `inspections/new` - Create inspection form
- [ ] `inspections/edit` - Edit inspection form
- [ ] `inspections/show` - Inspection detail view
- [ ] `inspections/review` - Single violation review
- [ ] `inspections/review_queue` - Queue-based violation review

### Partials (Convert as needed)
- [ ] `partials/image_upload`
- [ ] `partials/image_gallery`
- [ ] `partials/violation_form`
- [ ] `partials/violation_card`
- [ ] `partials/violations_summary`
- [ ] `partials/analysis_status`

### Layouts
- [ ] `layouts/public` - Public pages layout

## Notes

- Each page conversion should include CSRF protection for forms
- Use `internal/templ/pages/<section>/` for page components
- Use `internal/templ/shared/` for shared components (Flash, etc.)
- Handler methods follow pattern: `Show<Page>Templ` / `<Action>Templ`
- Wire up in handler's `RegisterTemplRoutes()` method
