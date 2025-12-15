# P0-003: Authentication Handlers & Pages - Implementation Plan

## Overview

This document provides a detailed implementation plan for authentication handlers (registration, login, logout) in the Lukaut application.

## Current State

### Completed Components

| Component | Status | Location |
|-----------|--------|----------|
| UserService interface | Complete | `/workspaces/lukaut/internal/service/user.go` |
| UserService implementation | Complete | `/workspaces/lukaut/internal/service/user.go` |
| Auth middleware | Complete | `/workspaces/lukaut/internal/middleware/auth.go` |
| Login template | Complete | `/workspaces/lukaut/web/templates/pages/auth/login.html` |
| Register template | Complete | `/workspaces/lukaut/web/templates/pages/auth/register.html` |
| Auth layout | Complete | `/workspaces/lukaut/web/templates/layouts/auth.html` |
| Flash message component | Complete | In auth layout |
| Renderer | Complete | `/workspaces/lukaut/internal/handler/renderer.go` |
| Error handling | Complete | `/workspaces/lukaut/internal/handler/error.go` |

### New Component Created

| Component | Status | Location |
|-----------|--------|----------|
| Auth handlers stub | Created | `/workspaces/lukaut/internal/handler/auth.go` |

---

## Handler Implementation Blueprint

### File: `/workspaces/lukaut/internal/handler/auth.go`

The auth handler file has been created with comprehensive documentation and method stubs. Key implementation details:

### 1. Handler Structure

```go
type AuthHandler struct {
    userService service.UserService
    renderer    *Renderer
    logger      *slog.Logger
    isSecure    bool  // true in production (HTTPS)
}
```

### 2. Routes to Implement

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /register | ShowRegister | Display registration form |
| POST | /register | Register | Process registration |
| GET | /login | ShowLogin | Display login form |
| POST | /login | Login | Process login |
| POST | /logout | Logout | Clear session and redirect |

### 3. Template Data Structure

```go
type AuthPageData struct {
    CurrentPath string            // For navigation highlighting
    CSRFToken   string            // Form protection token
    Form        map[string]string // Form values for repopulation
    Errors      map[string]string // Field-level validation errors
    Flash       *Flash            // Flash message (success/error/info)
    ReturnTo    string            // Post-login redirect URL
}

type Flash struct {
    Type    string // "success", "error", or "info"
    Message string
}
```

---

## Implementation Steps

### Step 1: Wire Up in main.go

Replace the current inline handlers with the AuthHandler:

```go
// In run() function, after initializing repository...

// Initialize repository
repo := repository.New(db)

// Initialize services
userService := service.NewUserService(repo, logger)

// Initialize middleware
isSecure := cfg.Env != "development"
authMw := middleware.NewAuthMiddleware(userService, logger, isSecure)

// Initialize handlers
authHandler := handler.NewAuthHandler(userService, renderer, logger, isSecure)

// Register routes
mux := http.NewServeMux()

// ... static files, health check ...

// Auth routes (public - no auth required)
authHandler.RegisterRoutes(mux)

// Protected routes example:
// authStack := middleware.Stack(authMw.WithUser, authMw.RequireUser)
// mux.Handle("GET /dashboard", authStack(http.HandlerFunc(dashboardHandler)))
```

### Step 2: Implement CSRF Protection (Priority: Medium)

The auth handler includes detailed documentation for implementing double-submit cookie CSRF protection. Key points:

1. Generate 32-byte random token using `crypto/rand`
2. Store token in cookie (NOT HttpOnly - form needs to read it)
3. Embed token in form using `csrfField` template function
4. On POST, compare cookie token with form token
5. Use `subtle.ConstantTimeCompare` for timing-safe comparison

Note: SameSite=Lax on session cookies already provides significant CSRF protection for modern browsers.

### Step 3: Add Authenticated User Checking (Priority: High)

Uncomment the user-already-logged-in check in ShowLogin and ShowRegister:

```go
// In ShowLogin and ShowRegister:
user := middleware.GetUser(r.Context())
if user != nil {
    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
    return
}
```

This requires wrapping the auth routes with `authMw.WithUser` middleware.

---

## Form Validation

### Registration Form Validation

| Field | Validation Rules |
|-------|-----------------|
| name | Required, trimmed |
| email | Required, valid format, normalized to lowercase |
| password | Required, minimum 8 characters |
| password_confirmation | Required, must match password |
| terms | Required checkbox (value = "on") |

### Login Form Validation

| Field | Validation Rules |
|-------|-----------------|
| email | Required, normalized to lowercase |
| password | Required |

### Error Display Pattern

1. Field-level errors go in `Errors` map (displayed below each field)
2. General errors go in `Flash` message (displayed at top of form)
3. On validation error, repopulate form fields (except passwords)
4. Service errors (like "email exists") shown as field error OR flash

---

## Security Considerations

### Password Handling
- Never log passwords
- Never repopulate password fields on error
- Hash passwords with bcrypt cost 12 (done in UserService)

### Session Management
- Session token is 32 bytes from crypto/rand
- Token stored in SHA-256 hash in database
- Cookie settings: HttpOnly, Secure (prod), SameSite=Lax
- Session duration: 7 days

### Login Security
- Generic error message: "Invalid email or password"
- Do NOT reveal if email exists in system
- UserService uses constant-time comparison
- Consider rate limiting (future enhancement)

### Redirect Safety
- Always validate return_to URLs with `isSafeRedirectURL()`
- Prevents open redirect vulnerabilities
- Only allows relative URLs starting with /

---

## Testing Checklist

### Manual Testing

- [ ] Register new user with valid data
- [ ] Register with duplicate email shows field error
- [ ] Register with invalid email shows field error
- [ ] Register with short password shows field error
- [ ] Register with mismatched passwords shows field error
- [ ] Register without accepting terms shows error
- [ ] Login with valid credentials redirects to dashboard
- [ ] Login with invalid email shows generic error
- [ ] Login with invalid password shows generic error
- [ ] Logout clears session and redirects to login
- [ ] Return_to parameter works after login
- [ ] Already logged-in user visiting /login redirects to dashboard

### Security Testing

- [ ] Session cookie has HttpOnly flag
- [ ] Session cookie has Secure flag (production)
- [ ] Session cookie has SameSite=Lax
- [ ] Password not visible in browser developer tools
- [ ] Invalid return_to URLs are rejected
- [ ] Logout actually invalidates session in database

---

## Template Integration Notes

### Existing Templates

The templates at `/workspaces/lukaut/web/templates/pages/auth/` are already set up to work with the AuthPageData structure:

**login.html expectations:**
- `.Form.Email` - Pre-filled email on error
- `.Errors.email` - Email field error
- `.Errors.password` - Password field error
- `.CSRFToken` - CSRF token for form
- `.Flash` - Flash message object

**register.html expectations:**
- `.Form.Name` - Pre-filled name on error
- `.Form.Email` - Pre-filled email on error
- `.Errors.name` - Name field error
- `.Errors.email` - Email field error
- `.Errors.password` - Password field error
- `.Errors.password_confirmation` - Confirmation error
- `.CSRFToken` - CSRF token for form
- `.Flash` - Flash message object

### Flash Message Display

The auth layout includes flash message rendering via `{{template "flash" .}}` which expects:

```go
type Flash struct {
    Type    string // "success", "error", or "info"
    Message string
}
```

The template handles all three types with appropriate styling.

---

## Remaining Work After This Implementation

1. **CSRF Protection** - Currently documented but commented out
2. **Rate Limiting** - Protect login/register from brute force
3. **Email Verification** - Verify email after registration (P0-004)
4. **Password Reset** - Forgot password flow (P0-006)
5. **Remember Me** - Extended session duration option

---

## Architecture Decision: Import Cycle Resolution

The middleware package imports handler for error response functions. To avoid an import cycle, the auth handler includes local copies of session cookie constants and helper functions.

Future refactor option: Create a shared `internal/session` package containing:
- Session cookie name and configuration
- Cookie setter/clearer functions
- Session token generation (if moved from service)

This would allow both handler and middleware to import from session without cycles.

---

## File Reference Summary

| File | Purpose |
|------|---------|
| `/workspaces/lukaut/internal/handler/auth.go` | Auth handler implementation |
| `/workspaces/lukaut/internal/service/user.go` | UserService interface and implementation |
| `/workspaces/lukaut/internal/middleware/auth.go` | Auth middleware (WithUser, RequireUser) |
| `/workspaces/lukaut/internal/handler/renderer.go` | Template renderer |
| `/workspaces/lukaut/internal/handler/error.go` | Error response helpers |
| `/workspaces/lukaut/internal/domain/user.go` | User, Session, RegisterParams types |
| `/workspaces/lukaut/web/templates/pages/auth/login.html` | Login template |
| `/workspaces/lukaut/web/templates/pages/auth/register.html` | Register template |
| `/workspaces/lukaut/web/templates/layouts/auth.html` | Auth page layout with flash |
| `/workspaces/lukaut/cmd/server/main.go` | Server entry point (needs updating) |
