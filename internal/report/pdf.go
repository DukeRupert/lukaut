package report

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/go-pdf/fpdf"
)

// =============================================================================
// PDF Generator
// =============================================================================

// PDFGenerator generates PDF reports from inspection data.
type PDFGenerator struct {
	// Page dimensions (A4 in mm)
	pageWidth  float64
	pageHeight float64
	margin     float64

	// Content area
	contentWidth float64
}

// NewPDFGenerator creates a new PDF generator with default settings.
func NewPDFGenerator() *PDFGenerator {
	margin := 15.0
	pageWidth := 210.0 // A4 width in mm
	return &PDFGenerator{
		pageWidth:    pageWidth,
		pageHeight:   297.0, // A4 height in mm
		margin:       margin,
		contentWidth: pageWidth - (2 * margin),
	}
}

// Format returns the output format of this generator.
func (g *PDFGenerator) Format() domain.ReportFormat {
	return domain.ReportFormatPDF
}

// Generate creates a PDF report and writes it to the provided writer.
func (g *PDFGenerator) Generate(ctx context.Context, data *domain.ReportData, w io.Writer) (int64, error) {
	pdf := fpdf.New("P", "mm", "A4", "")

	// Set document metadata
	pdf.SetTitle("Safety Inspection Report - "+data.InspectionTitle, true)
	pdf.SetAuthor(data.InspectorName, true)
	pdf.SetCreator("Lukaut Safety Inspection Platform", true)

	// Enable automatic page breaks with footer space
	pdf.SetAutoPageBreak(true, 20)

	// Set up footer on each page
	pdf.SetFooterFunc(func() {
		g.addFooter(pdf, data)
	})

	// Generate report sections
	g.addCoverPage(pdf, data)
	g.addExecutiveSummary(pdf, data)
	g.addSiteInformation(pdf, data)
	g.addFindings(pdf, data)

	// Only add appendix if there are regulations to show
	if g.hasRegulations(data) {
		g.addAppendix(pdf, data)
	}

	// Check for errors during generation
	if err := pdf.Error(); err != nil {
		return 0, fmt.Errorf("pdf generation error: %w", err)
	}

	// Write to buffer to count bytes
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return 0, fmt.Errorf("pdf output error: %w", err)
	}

	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// =============================================================================
// Cover Page
// =============================================================================

func (g *PDFGenerator) addCoverPage(pdf *fpdf.Fpdf, data *domain.ReportData) {
	pdf.AddPage()

	// Navy header bar
	r, gr, b := HexToRGB(BrandColors.Navy)
	pdf.SetFillColor(r, gr, b)
	pdf.Rect(0, 0, g.pageWidth, 70, "F")

	// Title
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 32)
	pdf.SetXY(g.margin, 25)
	pdf.Cell(0, 12, "Safety Inspection Report")

	// Subtitle with inspection title
	pdf.SetFont("Helvetica", "", 14)
	pdf.SetXY(g.margin, 42)
	pdf.Cell(0, 8, data.InspectionTitle)

	// Reset text color for body content
	r, gr, b = HexToRGB(BrandColors.TextDark)
	pdf.SetTextColor(r, gr, b)

	// Site information block
	pdf.SetXY(g.margin, 90)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(0, 8, "SITE")
	pdf.Ln(10)

	pdf.SetFont("Helvetica", "", 12)
	if data.SiteName != "" {
		pdf.Cell(0, 7, data.SiteName)
		pdf.Ln(7)
	}
	if data.SiteAddress != "" {
		pdf.Cell(0, 7, data.SiteAddress)
		pdf.Ln(7)
	}
	if data.SiteCity != "" || data.SiteState != "" {
		cityState := strings.TrimSpace(data.SiteCity + ", " + data.SiteState + " " + data.SitePostalCode)
		pdf.Cell(0, 7, cityState)
		pdf.Ln(7)
	}

	// Inspection date
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(0, 8, "INSPECTION DATE")
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 7, FormatDate(data.InspectionDate))

	// Inspector information
	pdf.Ln(15)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(0, 8, "INSPECTOR")
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 12)
	if data.InspectorName != "" {
		pdf.Cell(0, 7, data.InspectorName)
		pdf.Ln(7)
	}
	if data.InspectorCompany != "" {
		pdf.Cell(0, 7, data.InspectorCompany)
		pdf.Ln(7)
	}
	if data.InspectorLicense != "" {
		pdf.Cell(0, 7, "License: "+data.InspectorLicense)
		pdf.Ln(7)
	}

	// Client information (if available)
	if data.HasClient() {
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.Cell(0, 8, "CLIENT")
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "", 12)
		pdf.Cell(0, 7, data.ClientName)
		pdf.Ln(7)
		if data.ClientEmail != "" {
			pdf.Cell(0, 7, data.ClientEmail)
			pdf.Ln(7)
		}
		if data.ClientPhone != "" {
			pdf.Cell(0, 7, data.ClientPhone)
		}
	}
}

// =============================================================================
// Executive Summary
// =============================================================================

func (g *PDFGenerator) addExecutiveSummary(pdf *fpdf.Fpdf, data *domain.ReportData) {
	pdf.AddPage()

	// Section header
	g.addSectionHeader(pdf, "Executive Summary")

	// Violation counts table
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 8, "Violations Summary")
	pdf.Ln(10)

	counts := data.ViolationCountBySeverity()
	total := data.TotalViolations()

	// Table header
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(245, 245, 245)
	pdf.CellFormat(80, 8, "Severity", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, "Count", "1", 1, "C", true, 0, "")

	// Table rows
	pdf.SetFont("Helvetica", "", 10)
	severities := []domain.ViolationSeverity{
		domain.ViolationSeverityCritical,
		domain.ViolationSeveritySerious,
		domain.ViolationSeverityOther,
		domain.ViolationSeverityRecommendation,
	}

	for _, sev := range severities {
		count := counts[sev]
		if count > 0 || sev == domain.ViolationSeverityCritical || sev == domain.ViolationSeveritySerious {
			// Color indicator
			r, gr, b := HexToRGB(SeverityColor(sev))
			pdf.SetFillColor(r, gr, b)
			pdf.CellFormat(5, 8, "", "1", 0, "C", true, 0, "")
			pdf.SetFillColor(255, 255, 255)
			pdf.CellFormat(75, 8, SeverityLabel(sev), "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 8, fmt.Sprintf("%d", count), "1", 1, "C", false, 0, "")
		}
	}

	// Total row
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(245, 245, 245)
	pdf.CellFormat(80, 8, "Total", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, fmt.Sprintf("%d", total), "1", 1, "C", true, 0, "")

	// Conditions (if available)
	if data.WeatherConditions != "" || data.Temperature != "" {
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 8, "Conditions During Inspection")
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "", 10)

		if data.WeatherConditions != "" {
			pdf.Cell(0, 7, "Weather: "+data.WeatherConditions)
			pdf.Ln(7)
		}
		if data.Temperature != "" {
			pdf.Cell(0, 7, "Temperature: "+data.Temperature)
			pdf.Ln(7)
		}
	}

	// Inspector notes (if available)
	if data.InspectorNotes != "" {
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 8, "General Notes")
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "", 10)
		pdf.MultiCell(g.contentWidth, 6, data.InspectorNotes, "", "L", false)
	}
}

// =============================================================================
// Site Information
// =============================================================================

func (g *PDFGenerator) addSiteInformation(pdf *fpdf.Fpdf, data *domain.ReportData) {
	if !data.HasSite() && !data.HasClient() {
		return
	}

	pdf.AddPage()
	g.addSectionHeader(pdf, "Site & Client Information")

	// Site details
	if data.HasSite() {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 8, "Site Details")
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "", 10)

		g.addLabelValue(pdf, "Site Name", data.SiteName)
		g.addLabelValue(pdf, "Address", data.SiteFullAddress())
	}

	// Client details
	if data.HasClient() {
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 8, "Client Details")
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "", 10)

		g.addLabelValue(pdf, "Client Name", data.ClientName)
		if data.ClientEmail != "" {
			g.addLabelValue(pdf, "Email", data.ClientEmail)
		}
		if data.ClientPhone != "" {
			g.addLabelValue(pdf, "Phone", data.ClientPhone)
		}
	}

	// Inspector details
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 8, "Inspector Details")
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 10)

	g.addLabelValue(pdf, "Name", data.InspectorName)
	if data.InspectorCompany != "" {
		g.addLabelValue(pdf, "Company", data.InspectorCompany)
	}
	if data.InspectorLicense != "" {
		g.addLabelValue(pdf, "License #", data.InspectorLicense)
	}
	if data.InspectorEmail != "" {
		g.addLabelValue(pdf, "Email", data.InspectorEmail)
	}
	if data.InspectorPhone != "" {
		g.addLabelValue(pdf, "Phone", data.InspectorPhone)
	}
}

// =============================================================================
// Findings Section
// =============================================================================

func (g *PDFGenerator) addFindings(pdf *fpdf.Fpdf, data *domain.ReportData) {
	pdf.AddPage()
	g.addSectionHeader(pdf, "Inspection Findings")

	if len(data.Violations) == 0 {
		pdf.SetFont("Helvetica", "I", 11)
		pdf.Cell(0, 10, "No violations were identified during this inspection.")
		return
	}

	for i, violation := range data.Violations {
		// Check if we need a new page (leave room for at least basic violation info)
		if pdf.GetY() > 230 {
			pdf.AddPage()
		}

		g.addViolation(pdf, data, violation, i+1)

		// Add spacing between violations
		if i < len(data.Violations)-1 {
			pdf.Ln(8)
			// Draw separator line
			r, gr, b := HexToRGB(BrandColors.Border)
			pdf.SetDrawColor(r, gr, b)
			pdf.Line(g.margin, pdf.GetY(), g.pageWidth-g.margin, pdf.GetY())
			pdf.Ln(8)
		}
	}
}

func (g *PDFGenerator) addViolation(pdf *fpdf.Fpdf, data *domain.ReportData, violation domain.ReportViolation, number int) {
	// Violation header with severity indicator
	r, gr, b := HexToRGB(SeverityColor(violation.Severity))
	pdf.SetFillColor(r, gr, b)
	pdf.Rect(g.margin, pdf.GetY(), 4, 8, "F")

	pdf.SetX(g.margin + 8)
	pdf.SetFont("Helvetica", "B", 12)
	r, gr, b = HexToRGB(BrandColors.TextDark)
	pdf.SetTextColor(r, gr, b)
	pdf.Cell(0, 8, fmt.Sprintf("Finding #%d", number))
	pdf.Ln(10)

	// Severity badge
	pdf.SetX(g.margin + 8)
	pdf.SetFont("Helvetica", "", 10)
	r, gr, b = HexToRGB(SeverityColor(violation.Severity))
	pdf.SetTextColor(r, gr, b)
	pdf.Cell(0, 6, "Severity: "+SeverityLabel(violation.Severity))
	pdf.Ln(8)

	// Reset text color
	r, gr, b = HexToRGB(BrandColors.TextDark)
	pdf.SetTextColor(r, gr, b)

	// Description
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, "Description:")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(g.contentWidth, 5, violation.Description, "", "L", false)
	pdf.Ln(4)

	// Primary regulation citation
	if reg := violation.PrimaryRegulation(); reg != nil {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(0, 6, "OSHA Regulation:")
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "", 10)

		// Standard number in orange
		r, gr, b = HexToRGB(BrandColors.SafetyOrange)
		pdf.SetTextColor(r, gr, b)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(0, 5, reg.StandardNumber)
		pdf.Ln(5)

		// Title
		r, gr, b = HexToRGB(BrandColors.TextDark)
		pdf.SetTextColor(r, gr, b)
		pdf.SetFont("Helvetica", "", 10)
		if reg.Title != "" {
			pdf.Cell(0, 5, reg.Title)
			pdf.Ln(5)
		}
		if reg.Category != "" {
			r, gr, b = HexToRGB(BrandColors.TextMuted)
			pdf.SetTextColor(r, gr, b)
			pdf.Cell(0, 5, "Category: "+reg.Category)
			pdf.Ln(5)
		}

		// Reset text color
		r, gr, b = HexToRGB(BrandColors.TextDark)
		pdf.SetTextColor(r, gr, b)
		pdf.Ln(2)
	}

	// Inspector notes
	if violation.InspectorNotes != "" {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(0, 6, "Inspector Notes:")
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "I", 10)
		pdf.MultiCell(g.contentWidth, 5, violation.InspectorNotes, "", "L", false)
	}
}

// =============================================================================
// Appendix
// =============================================================================

func (g *PDFGenerator) addAppendix(pdf *fpdf.Fpdf, data *domain.ReportData) {
	pdf.AddPage()
	g.addSectionHeader(pdf, "Appendix: Regulation Reference")

	// Collect unique regulations
	regulations := make(map[string]domain.ReportRegulation)
	for _, v := range data.Violations {
		for _, r := range v.Regulations {
			if _, exists := regulations[r.StandardNumber]; !exists {
				regulations[r.StandardNumber] = r
			}
		}
	}

	if len(regulations) == 0 {
		pdf.SetFont("Helvetica", "I", 10)
		pdf.Cell(0, 8, "No regulations cited in this report.")
		return
	}

	pdf.SetFont("Helvetica", "", 10)
	r, gr, b := HexToRGB(BrandColors.TextDark)
	pdf.SetTextColor(r, gr, b)

	for stdNum, reg := range regulations {
		// Check for page break
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}

		// Standard number header
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 8, stdNum)
		pdf.Ln(8)

		// Title
		if reg.Title != "" {
			pdf.SetFont("Helvetica", "B", 10)
			pdf.Cell(0, 6, reg.Title)
			pdf.Ln(6)
		}

		// Category
		if reg.Category != "" {
			pdf.SetFont("Helvetica", "I", 9)
			r, gr, b = HexToRGB(BrandColors.TextMuted)
			pdf.SetTextColor(r, gr, b)
			pdf.Cell(0, 5, "Category: "+reg.Category)
			pdf.Ln(6)
			r, gr, b = HexToRGB(BrandColors.TextDark)
			pdf.SetTextColor(r, gr, b)
		}

		// Full text (truncated if very long)
		if reg.FullText != "" {
			pdf.SetFont("Helvetica", "", 9)
			fullText := reg.FullText
			if len(fullText) > 1000 {
				fullText = fullText[:1000] + "..."
			}
			pdf.MultiCell(g.contentWidth, 5, fullText, "", "L", false)
		}

		pdf.Ln(8)

		// Separator
		r, gr, b = HexToRGB(BrandColors.Border)
		pdf.SetDrawColor(r, gr, b)
		pdf.Line(g.margin, pdf.GetY(), g.pageWidth-g.margin, pdf.GetY())
		pdf.Ln(8)
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

func (g *PDFGenerator) addSectionHeader(pdf *fpdf.Fpdf, title string) {
	// Draw navy underline
	r, gr, b := HexToRGB(BrandColors.Navy)
	pdf.SetDrawColor(r, gr, b)
	pdf.SetLineWidth(0.5)

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(r, gr, b)
	pdf.Cell(0, 10, title)
	pdf.Ln(12)

	pdf.Line(g.margin, pdf.GetY(), g.pageWidth-g.margin, pdf.GetY())
	pdf.Ln(10)

	// Reset text color
	r, gr, b = HexToRGB(BrandColors.TextDark)
	pdf.SetTextColor(r, gr, b)
}

func (g *PDFGenerator) addLabelValue(pdf *fpdf.Fpdf, label, value string) {
	if value == "" {
		return
	}
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(40, 6, label+":")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(g.contentWidth-40, 6, value, "", "L", false)
}

func (g *PDFGenerator) addFooter(pdf *fpdf.Fpdf, data *domain.ReportData) {
	pdf.SetY(-15)

	// Draw separator line
	r, gr, b := HexToRGB(BrandColors.Border)
	pdf.SetDrawColor(r, gr, b)
	pdf.Line(g.margin, pdf.GetY()-3, g.pageWidth-g.margin, pdf.GetY()-3)

	// Footer text
	r, gr, b = HexToRGB(BrandColors.TextMuted)
	pdf.SetTextColor(r, gr, b)
	pdf.SetFont("Helvetica", "", 8)

	// Left: generation date
	pdf.Cell(0, 10, "Generated: "+FormatDateTime(data.GeneratedAt))

	// Right: page number
	pdf.SetX(-g.margin - 30)
	pdf.CellFormat(30, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
}

func (g *PDFGenerator) hasRegulations(data *domain.ReportData) bool {
	for _, v := range data.Violations {
		if len(v.Regulations) > 0 {
			return true
		}
	}
	return false
}
