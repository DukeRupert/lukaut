// Package domain contains core business types and interfaces.
//
// This file defines the Report domain types and data structures for
// generating construction safety inspection reports in PDF and DOCX formats.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Report Format
// =============================================================================

// ReportFormat represents the output format of a report.
type ReportFormat string

const (
	// ReportFormatPDF generates a PDF document.
	ReportFormatPDF ReportFormat = "pdf"

	// ReportFormatDOCX generates a Microsoft Word document.
	ReportFormatDOCX ReportFormat = "docx"
)

// String returns the string representation of the format.
func (f ReportFormat) String() string {
	return string(f)
}

// IsValid returns true if the format is a recognized value.
func (f ReportFormat) IsValid() bool {
	switch f {
	case ReportFormatPDF, ReportFormatDOCX:
		return true
	}
	return false
}

// ContentType returns the MIME content type for the format.
func (f ReportFormat) ContentType() string {
	switch f {
	case ReportFormatPDF:
		return "application/pdf"
	case ReportFormatDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

// FileExtension returns the file extension for the format.
func (f ReportFormat) FileExtension() string {
	return string(f)
}

// =============================================================================
// Report Domain Type
// =============================================================================

// Report represents a generated inspection report stored in the database.
type Report struct {
	ID             uuid.UUID // Unique identifier
	InspectionID   uuid.UUID // Inspection this report was generated from
	UserID         uuid.UUID // User who generated the report
	PDFStorageKey  string    // Storage key for PDF file (empty if not generated)
	DOCXStorageKey string    // Storage key for DOCX file (empty if not generated)
	ViolationCount int       // Number of violations included in report
	GeneratedAt    time.Time // When report was generated
}

// HasPDF returns true if this report has a PDF version.
func (r *Report) HasPDF() bool {
	return r.PDFStorageKey != ""
}

// HasDOCX returns true if this report has a DOCX version.
func (r *Report) HasDOCX() bool {
	return r.DOCXStorageKey != ""
}

// =============================================================================
// Report Data Aggregates (for generation)
// =============================================================================

// ReportData aggregates all data needed to generate a report.
// This struct is populated by the job handler before passing to generators.
type ReportData struct {
	// Inspector/User information
	InspectorName    string // Inspector's display name
	InspectorCompany string // Company/business name
	InspectorLicense string // Professional license number
	InspectorEmail   string // Contact email
	InspectorPhone   string // Contact phone
	InspectorAddress string // Business address (formatted)
	InspectorLogoURL string // URL to company logo (for embedding)

	// Inspection details
	InspectionID      uuid.UUID // Inspection ID
	InspectionTitle   string    // Inspection title
	InspectionDate    time.Time // Date inspection was conducted
	WeatherConditions string    // Weather during inspection
	Temperature       string    // Temperature during inspection
	InspectorNotes    string    // General notes from inspector

	// Site information
	SiteName       string // Site name
	SiteAddress    string // Street address
	SiteCity       string // City
	SiteState      string // State
	SitePostalCode string // Postal code

	// Client information (via site)
	ClientName  string // Client company name
	ClientEmail string // Client contact email
	ClientPhone string // Client contact phone

	// Violations with regulations
	Violations []ReportViolation

	// Metadata
	GeneratedAt time.Time // When report is being generated
}

// TotalViolations returns the total number of violations.
func (d *ReportData) TotalViolations() int {
	return len(d.Violations)
}

// ViolationCountBySeverity returns counts grouped by severity level.
func (d *ReportData) ViolationCountBySeverity() map[ViolationSeverity]int {
	counts := make(map[ViolationSeverity]int)
	for _, v := range d.Violations {
		counts[v.Severity]++
	}
	return counts
}

// HasSite returns true if site information is available.
func (d *ReportData) HasSite() bool {
	return d.SiteName != "" || d.SiteAddress != ""
}

// HasClient returns true if client information is available.
func (d *ReportData) HasClient() bool {
	return d.ClientName != ""
}

// SiteFullAddress returns the complete formatted site address.
func (d *ReportData) SiteFullAddress() string {
	if d.SiteAddress == "" {
		return ""
	}
	addr := d.SiteAddress
	if d.SiteCity != "" || d.SiteState != "" || d.SitePostalCode != "" {
		addr += "\n"
		if d.SiteCity != "" {
			addr += d.SiteCity
		}
		if d.SiteState != "" {
			if d.SiteCity != "" {
				addr += ", "
			}
			addr += d.SiteState
		}
		if d.SitePostalCode != "" {
			addr += " " + d.SitePostalCode
		}
	}
	return addr
}

// =============================================================================
// Report Violation
// =============================================================================

// ReportViolation contains violation data formatted for reports.
type ReportViolation struct {
	Number         int               // Sequential number in report
	Description    string            // Violation description
	Severity       ViolationSeverity // Severity level
	InspectorNotes string            // Additional inspector notes
	ThumbnailURL   string            // URL to thumbnail image (presigned)
	Regulations    []ReportRegulation
}

// PrimaryRegulation returns the primary regulation for this violation, if any.
func (v *ReportViolation) PrimaryRegulation() *ReportRegulation {
	for i := range v.Regulations {
		if v.Regulations[i].IsPrimary {
			return &v.Regulations[i]
		}
	}
	if len(v.Regulations) > 0 {
		return &v.Regulations[0]
	}
	return nil
}

// HasRegulations returns true if this violation has linked regulations.
func (v *ReportViolation) HasRegulations() bool {
	return len(v.Regulations) > 0
}

// =============================================================================
// Report Regulation
// =============================================================================

// ReportRegulation contains regulation data for reports.
type ReportRegulation struct {
	StandardNumber string  // OSHA standard number (e.g., "1926.501(b)(1)")
	Title          string  // Regulation title
	Category       string  // Category (e.g., "Fall Protection")
	FullText       string  // Complete regulation text (for appendix)
	IsPrimary      bool    // Whether this is the primary regulation
	RelevanceScore float64 // AI relevance score (0.0-1.0)
}

// Citation returns a formatted citation string.
func (r *ReportRegulation) Citation() string {
	if r.Title != "" {
		return r.StandardNumber + " - " + r.Title
	}
	return r.StandardNumber
}
