package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateFuncs returns a FuncMap with custom template functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// Math functions
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"mult": func(a, b int) int {
			return a * b
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},

		// Date/Time functions
		"year": func() int {
			return time.Now().Year()
		},
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"formatDateISO": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"timeAgo": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			now := time.Now()
			diff := now.Sub(t)

			switch {
			case diff < time.Minute:
				return "just now"
			case diff < time.Hour:
				mins := int(diff.Minutes())
				if mins == 1 {
					return "1 minute ago"
				}
				return fmt.Sprintf("%d minutes ago", mins)
			case diff < 24*time.Hour:
				hours := int(diff.Hours())
				if hours == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", hours)
			case diff < 7*24*time.Hour:
				days := int(diff.Hours() / 24)
				if days == 1 {
					return "yesterday"
				}
				return fmt.Sprintf("%d days ago", days)
			case diff < 30*24*time.Hour:
				weeks := int(diff.Hours() / 24 / 7)
				if weeks == 1 {
					return "1 week ago"
				}
				return fmt.Sprintf("%d weeks ago", weeks)
			default:
				return t.Format("Jan 2, 2006")
			}
		},

		// String functions
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(v interface{}) string {
			s := fmt.Sprint(v)
			return cases.Title(language.English).String(s)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		// JSON encoding for safe JavaScript embedding
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS(`""`)
			}
			return template.JS(b)
		},

		// Conditional/Logic functions
		"ternary": func(condition bool, trueVal, falseVal interface{}) interface{} {
			if condition {
				return trueVal
			}
			return falseVal
		},
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" || val == 0 {
				return defaultVal
			}
			return val
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"isset": func(m map[string]interface{}, key string) bool {
			_, ok := m[key]
			return ok
		},

		// Collection functions
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
		"pageRange": func(currentPage, totalPages int) []int {
			// Show max 7 page numbers
			maxPages := 7
			if totalPages <= maxPages {
				result := []int{}
				for i := 1; i <= totalPages; i++ {
					result = append(result, i)
				}
				return result
			}

			// Calculate range around current page
			start := currentPage - 3
			end := currentPage + 3

			// Adjust if at beginning
			if start < 1 {
				start = 1
				end = maxPages
			}

			// Adjust if at end
			if end > totalPages {
				end = totalPages
				start = totalPages - maxPages + 1
			}

			result := []int{}
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},

		// HTML rendering functions
		"html": func(s string) template.HTML {
			return template.HTML(s)
		},
		"attr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},

		// UUID functions
		"uuidString": func(u uuid.UUID) string {
			return u.String()
		},

		// Form helpers
		"csrfField": func(token string) template.HTML {
			return template.HTML(fmt.Sprintf(`<input type="hidden" name="csrf_token" value="%s">`, template.HTMLEscapeString(token)))
		},

		// Status/badge helpers for inspections
		// These functions accept interface{} to handle custom string types like domain.InspectionStatus
		"statusColor": func(status interface{}) string {
			s := fmt.Sprint(status)
			switch s {
			case "draft":
				return "bg-clay/20 text-clay"
			case "analyzing":
				return "bg-gold/20 text-forest"
			case "review":
				return "bg-gold text-forest"
			case "completed":
				return "bg-forest text-white"
			default:
				return "bg-gray-100 text-gray-600"
			}
		},
		"severityColor": func(severity interface{}) string {
			s := fmt.Sprint(severity)
			switch s {
			case "critical":
				return "bg-red-100 text-red-800"
			case "serious":
				return "bg-orange-100 text-orange-800"
			case "other":
				return "bg-yellow-100 text-yellow-800"
			case "recommendation":
				return "bg-blue-100 text-blue-800"
			default:
				return "bg-gray-100 text-gray-600"
			}
		},
		"confidenceColor": func(confidence interface{}) string {
			s := fmt.Sprint(confidence)
			switch s {
			case "high":
				return "text-forest"
			case "medium":
				return "text-gold"
			case "low":
				return "text-clay"
			default:
				return "text-gray-500"
			}
		},
	}
}
