package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// Renderer manages template parsing and rendering with isolated template sets.
// It supports two layouts:
//   - "auth" layout for unauthenticated pages (login, register, password reset)
//   - "app" layout for authenticated pages (dashboard, inspections, etc.)
//
// Templates are organized as:
//   - layouts/auth.html, layouts/app.html - base layouts
//   - components/*.html - reusable components (shared across layouts)
//   - partials/*.html - standalone fragments for htmx responses
//   - pages/auth/*.html - auth pages (use auth layout)
//   - pages/*.html - app pages (use app layout)
type Renderer struct {
	templates map[string]*template.Template
	logger    *slog.Logger
	isDev     bool
	mu        sync.RWMutex

	// For dev mode hot-reload
	templatesDir string
}

// RendererConfig holds configuration for the renderer.
type RendererConfig struct {
	TemplatesDir string
	Logger       *slog.Logger
	IsDev        bool
}

// NewRenderer creates a new template renderer.
func NewRenderer(cfg RendererConfig) (*Renderer, error) {
	r := &Renderer{
		templates:    make(map[string]*template.Template),
		logger:       cfg.Logger,
		isDev:        cfg.IsDev,
		templatesDir: cfg.TemplatesDir,
	}

	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// NewRendererFromFS creates a renderer from an embedded filesystem.
func NewRendererFromFS(fsys fs.FS, logger *slog.Logger) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		logger:    logger,
		isDev:     false,
	}

	if err := r.loadTemplatesFromFS(fsys); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) loadTemplates() error {
	templatesDir := r.templatesDir

	// Get component templates (shared across layouts) - recursively from all subdirs
	var componentFiles []string
	componentsDir := filepath.Join(templatesDir, "components")
	err := filepath.WalkDir(componentsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			componentFiles = append(componentFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk components dir: %w", err)
	}

	// Get partial templates (standalone fragments for htmx)
	partialsPattern := filepath.Join(templatesDir, "partials", "*.html")
	partialFiles, err := filepath.Glob(partialsPattern)
	if err != nil {
		return fmt.Errorf("failed to glob partials: %w", err)
	}

	// Parse each partial as a standalone template
	for _, partial := range partialFiles {
		partialTmpl, err := template.New("").Funcs(TemplateFuncs()).ParseFiles(partial)
		if err != nil {
			return fmt.Errorf("failed to parse partial %s: %w", partial, err)
		}

		// Store with base name as key (e.g., "flash" for "flash.html")
		partialName := filepath.Base(partial)
		partialName = strings.TrimSuffix(partialName, filepath.Ext(partialName))
		r.templates["partial/"+partialName] = partialTmpl
	}

	// Parse public layout (for marketing pages)
	publicLayoutPath := filepath.Join(templatesDir, "layouts", "public.html")
	publicBaseTmpl, err := template.New("public").Funcs(TemplateFuncs()).ParseFiles(publicLayoutPath)
	if err != nil {
		return fmt.Errorf("failed to parse public layout: %w", err)
	}

	// Parse components into public layout
	if len(componentFiles) > 0 {
		publicBaseTmpl, err = publicBaseTmpl.ParseFiles(componentFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse components into public layout: %w", err)
		}
	}

	// Parse partials into public layout
	if len(partialFiles) > 0 {
		publicBaseTmpl, err = publicBaseTmpl.ParseFiles(partialFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse partials into public layout: %w", err)
		}
	}

	// Parse auth layout
	authLayoutPath := filepath.Join(templatesDir, "layouts", "auth.html")
	authBaseTmpl, err := template.New("auth").Funcs(TemplateFuncs()).ParseFiles(authLayoutPath)
	if err != nil {
		return fmt.Errorf("failed to parse auth layout: %w", err)
	}

	// Parse components into auth layout
	if len(componentFiles) > 0 {
		authBaseTmpl, err = authBaseTmpl.ParseFiles(componentFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse components into auth layout: %w", err)
		}
	}

	// Parse partials into auth layout (so pages can use {{template "partial_name"}})
	if len(partialFiles) > 0 {
		authBaseTmpl, err = authBaseTmpl.ParseFiles(partialFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse partials into auth layout: %w", err)
		}
	}

	// Parse app layout
	appLayoutPath := filepath.Join(templatesDir, "layouts", "app.html")
	appBaseTmpl, err := template.New("app").Funcs(TemplateFuncs()).ParseFiles(appLayoutPath)
	if err != nil {
		return fmt.Errorf("failed to parse app layout: %w", err)
	}

	// Parse components into app layout
	if len(componentFiles) > 0 {
		appBaseTmpl, err = appBaseTmpl.ParseFiles(componentFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse components into app layout: %w", err)
		}
	}

	// Parse partials into app layout
	if len(partialFiles) > 0 {
		appBaseTmpl, err = appBaseTmpl.ParseFiles(partialFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse partials into app layout: %w", err)
		}
	}

	// Parse public pages (home, pricing, etc.)
	publicPages, err := filepath.Glob(filepath.Join(templatesDir, "pages", "public", "*.html"))
	if err != nil {
		return fmt.Errorf("failed to glob public pages: %w", err)
	}

	for _, page := range publicPages {
		pageTmpl, err := publicBaseTmpl.Clone()
		if err != nil {
			return fmt.Errorf("failed to clone public template for %s: %w", page, err)
		}

		pageTmpl, err = pageTmpl.ParseFiles(page)
		if err != nil {
			return fmt.Errorf("failed to parse public page %s: %w", page, err)
		}

		// Store as "public/home", "public/pricing", etc.
		pageName := filepath.Base(page)
		pageName = strings.TrimSuffix(pageName, filepath.Ext(pageName))
		r.templates["public/"+pageName] = pageTmpl
	}

	// Parse auth pages (login, register, forgot-password, etc.)
	authPages, err := filepath.Glob(filepath.Join(templatesDir, "pages", "auth", "*.html"))
	if err != nil {
		return fmt.Errorf("failed to glob auth pages: %w", err)
	}

	for _, page := range authPages {
		pageTmpl, err := authBaseTmpl.Clone()
		if err != nil {
			return fmt.Errorf("failed to clone auth template for %s: %w", page, err)
		}

		pageTmpl, err = pageTmpl.ParseFiles(page)
		if err != nil {
			return fmt.Errorf("failed to parse auth page %s: %w", page, err)
		}

		// Store as "auth/login", "auth/register", etc.
		pageName := filepath.Base(page)
		pageName = strings.TrimSuffix(pageName, filepath.Ext(pageName))
		r.templates["auth/"+pageName] = pageTmpl
	}

	// Parse app pages (dashboard, etc. - root level pages use app layout)
	appPages, err := filepath.Glob(filepath.Join(templatesDir, "pages", "*.html"))
	if err != nil {
		return fmt.Errorf("failed to glob app pages: %w", err)
	}

	for _, page := range appPages {
		pageTmpl, err := appBaseTmpl.Clone()
		if err != nil {
			return fmt.Errorf("failed to clone app template for %s: %w", page, err)
		}

		pageTmpl, err = pageTmpl.ParseFiles(page)
		if err != nil {
			return fmt.Errorf("failed to parse app page %s: %w", page, err)
		}

		// Store as "dashboard", "settings", etc.
		pageName := filepath.Base(page)
		pageName = strings.TrimSuffix(pageName, filepath.Ext(pageName))
		r.templates[pageName] = pageTmpl
	}

	// Parse nested app pages (inspections/*, reports/*, etc.)
	nestedDirs := []string{"inspections", "reports", "settings", "regulations"}
	for _, dir := range nestedDirs {
		pattern := filepath.Join(templatesDir, "pages", dir, "*.html")
		nestedPages, err := filepath.Glob(pattern)
		if err != nil {
			continue // Directory might not exist yet
		}

		for _, page := range nestedPages {
			pageTmpl, err := appBaseTmpl.Clone()
			if err != nil {
				return fmt.Errorf("failed to clone app template for %s: %w", page, err)
			}

			pageTmpl, err = pageTmpl.ParseFiles(page)
			if err != nil {
				return fmt.Errorf("failed to parse page %s: %w", page, err)
			}

			// Store as "inspections/index", "inspections/show", etc.
			pageName := filepath.Base(page)
			pageName = strings.TrimSuffix(pageName, filepath.Ext(pageName))
			r.templates[dir+"/"+pageName] = pageTmpl
		}
	}

	r.logger.Info("templates loaded", "count", len(r.templates))
	return nil
}

func (r *Renderer) loadTemplatesFromFS(fsys fs.FS) error {
	// Similar to loadTemplates but using fs.FS
	// Implementation for embedded templates in production
	// For now, just return nil - can be implemented when needed
	return fmt.Errorf("loadTemplatesFromFS not implemented yet")
}

// Reload reloads all templates from disk. Useful for development.
func (r *Renderer) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.templates = make(map[string]*template.Template)
	return r.loadTemplates()
}

// Execute returns the named template for manual execution.
func (r *Renderer) Execute(name string) (*template.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return tmpl, nil
}

// Render renders a template to an io.Writer.
func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	// In dev mode, reload templates on each request
	if r.isDev {
		if err := r.Reload(); err != nil {
			return fmt.Errorf("template reload failed: %w", err)
		}
	}

	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	// Determine base template name based on template path
	execName := r.getBaseTemplateName(name)

	return tmpl.ExecuteTemplate(w, execName, data)
}

// RenderHTML renders a template and returns the HTML as a string.
func (r *Renderer) RenderHTML(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := r.Render(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderHTTP renders a template directly to an http.ResponseWriter.
func (r *Renderer) RenderHTTP(w http.ResponseWriter, name string, data interface{}) {
	// In dev mode, reload templates on each request
	if r.isDev {
		if err := r.Reload(); err != nil {
			r.logger.Error("template reload failed", "error", err)
			http.Error(w, "Template reload failed", http.StatusInternalServerError)
			return
		}
	}

	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		r.logger.Error("template not found", "name", name)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Determine base template name
	execName := r.getBaseTemplateName(name)

	// Render to buffer first to catch errors before writing headers
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, execName, data); err != nil {
		r.logger.Error("template execution failed", "name", name, "error", err)
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// RenderPartial renders a partial template (for htmx responses).
// The partial file should contain {{define "name"}}...{{end}} where name matches the file name.
func (r *Renderer) RenderPartial(w http.ResponseWriter, name string, data interface{}) {
	fullName := "partial/" + name

	r.mu.RLock()
	tmpl, ok := r.templates[fullName]
	r.mu.RUnlock()

	if !ok {
		r.logger.Error("partial template not found", "name", name)
		http.Error(w, "Partial not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Execute the named template within the partial file
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		r.logger.Error("partial execution failed", "name", name, "error", err)
	}
}

// getBaseTemplateName determines which base template to execute.
func (r *Renderer) getBaseTemplateName(name string) string {
	switch {
	case strings.HasPrefix(name, "public/"):
		return "public"
	case strings.HasPrefix(name, "auth/"):
		return "auth"
	case strings.HasPrefix(name, "partial/"):
		// Partials execute the file's base template name
		return filepath.Base(name) + ".html"
	default:
		return "app"
	}
}

// ListTemplates returns a list of all loaded template names.
// Useful for debugging.
func (r *Renderer) ListTemplates() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// ToastData holds data for rendering a toast notification.
type ToastData struct {
	Type        string // success, error, warning, info
	Title       string // optional
	Message     string
	AutoDismiss int // seconds, default 5
}

// RenderHTTPWithToast renders a template and appends an OOB toast notification.
// This is useful for htmx responses that need to show feedback.
func (r *Renderer) RenderHTTPWithToast(w http.ResponseWriter, name string, data interface{}, toast ToastData) {
	// In dev mode, reload templates on each request
	if r.isDev {
		if err := r.Reload(); err != nil {
			r.logger.Error("template reload failed", "error", err)
			http.Error(w, "Template reload failed", http.StatusInternalServerError)
			return
		}
	}

	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		r.logger.Error("template not found", "name", name)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Determine base template name
	execName := r.getBaseTemplateName(name)

	// Render main content to buffer
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, execName, data); err != nil {
		r.logger.Error("template execution failed", "name", name, "error", err)
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}

	// Render toast OOB
	if toast.AutoDismiss == 0 {
		toast.AutoDismiss = 5
	}
	if toast.Type == "" {
		toast.Type = "info"
	}

	toastHTML := r.renderToastOOB(toast)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
	w.Write([]byte(toastHTML))
}

// renderToastOOB generates the toast OOB HTML.
func (r *Renderer) renderToastOOB(toast ToastData) string {
	icon := r.getToastIcon(toast.Type)

	titleHTML := ""
	if toast.Title != "" {
		titleHTML = fmt.Sprintf(`<p class="text-sm font-medium text-[var(--color-text)]">%s</p>`, toast.Title)
	}

	messageClass := "text-sm text-[var(--color-text-secondary)]"
	if toast.Title != "" {
		messageClass = "mt-1 " + messageClass
	}

	return fmt.Sprintf(`<div hx-swap-oob="beforeend:#toast-container">
  <div x-data="{ show: true, init() { setTimeout(() => { this.show = false; setTimeout(() => this.$el.remove(), 300) }, %d000) } }"
       x-show="show"
       x-transition:enter="transition ease-out duration-300"
       x-transition:enter-start="opacity-0 translate-x-4"
       x-transition:enter-end="opacity-100 translate-x-0"
       x-transition:leave="transition ease-in duration-200"
       x-transition:leave-start="opacity-100 translate-x-0"
       x-transition:leave-end="opacity-0 translate-x-4"
       class="pointer-events-auto w-full max-w-sm overflow-hidden rounded-lg bg-white shadow-lg ring-1 ring-[var(--color-zinc-950)]/5">
    <div class="p-4">
      <div class="flex items-start">
        <div class="flex-shrink-0">
          %s
        </div>
        <div class="ml-3 w-0 flex-1 pt-0.5">
          %s
          <p class="%s">%s</p>
        </div>
        <div class="ml-4 flex flex-shrink-0">
          <button type="button"
                  @click="show = false; setTimeout(() => $el.closest('[x-data]').remove(), 300)"
                  class="inline-flex rounded-md text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] focus:ring-offset-2">
            <span class="sr-only">Close</span>
            <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</div>`, toast.AutoDismiss, icon, titleHTML, messageClass, toast.Message)
}

func (r *Renderer) getToastIcon(toastType string) string {
	switch toastType {
	case "success":
		return `<svg class="h-6 w-6 text-[var(--color-success)]" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>`
	case "error":
		return `<svg class="h-6 w-6 text-[var(--color-danger)]" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>`
	case "warning":
		return `<svg class="h-6 w-6 text-[var(--color-warning)]" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" /></svg>`
	default: // info
		return `<svg class="h-6 w-6 text-[var(--color-info)]" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" /></svg>`
	}
}
