package report

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/unidoc/unioffice/color"
	"github.com/unidoc/unioffice/document"
	"github.com/unidoc/unioffice/measurement"
)

// =============================================================================
// DOCX Generator
// =============================================================================

// DOCXGenerator generates DOCX reports from inspection data.
type DOCXGenerator struct{}

// NewDOCXGenerator creates a new DOCX generator.
func NewDOCXGenerator() *DOCXGenerator {
	return &DOCXGenerator{}
}

// Format returns the output format of this generator.
func (g *DOCXGenerator) Format() domain.ReportFormat {
	return domain.ReportFormatDOCX
}

// Generate creates a DOCX report and writes it to the provided writer.
func (g *DOCXGenerator) Generate(ctx context.Context, data *domain.ReportData, w io.Writer) (int64, error) {
	doc := document.New()
	defer doc.Close()

	// Set document properties
	props := doc.CoreProperties
	props.SetTitle("Safety Inspection Report - " + data.InspectionTitle)
	props.SetAuthor(data.InspectorName)

	// Generate report sections
	g.addCoverSection(doc, data)
	g.addExecutiveSummary(doc, data)
	g.addSiteInformation(doc, data)
	g.addFindings(doc, data)

	if g.hasRegulations(data) {
		g.addAppendix(doc, data)
	}

	// Write to buffer to count bytes
	var buf bytes.Buffer
	if err := doc.Save(&buf); err != nil {
		return 0, fmt.Errorf("docx save error: %w", err)
	}

	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// =============================================================================
// Cover Section
// =============================================================================

func (g *DOCXGenerator) addCoverSection(doc *document.Document, data *domain.ReportData) {
	// Main title
	title := doc.AddParagraph()
	titleRun := title.AddRun()
	titleRun.Properties().SetBold(true)
	titleRun.Properties().SetSize(32 * measurement.Point)
	titleRun.Properties().SetColor(color.RGB(30, 58, 95)) // Navy
	titleRun.AddText("Safety Inspection Report")
	title.Properties().SetSpacing(0, 20*measurement.Point)

	// Inspection title
	subtitle := doc.AddParagraph()
	subtitleRun := subtitle.AddRun()
	subtitleRun.Properties().SetSize(14 * measurement.Point)
	subtitleRun.AddText(data.InspectionTitle)
	subtitle.Properties().SetSpacing(0, 30*measurement.Point)

	// Site information
	g.addLabeledSection(doc, "SITE", func() {
		if data.SiteName != "" {
			g.addTextLine(doc, data.SiteName, false)
		}
		if data.SiteAddress != "" {
			g.addTextLine(doc, data.SiteAddress, false)
		}
		if data.SiteCity != "" || data.SiteState != "" {
			cityState := data.SiteCity
			if data.SiteState != "" {
				if cityState != "" {
					cityState += ", "
				}
				cityState += data.SiteState
			}
			if data.SitePostalCode != "" {
				cityState += " " + data.SitePostalCode
			}
			g.addTextLine(doc, cityState, false)
		}
	})

	// Inspection date
	g.addLabeledSection(doc, "INSPECTION DATE", func() {
		g.addTextLine(doc, FormatDate(data.InspectionDate), false)
	})

	// Inspector information
	g.addLabeledSection(doc, "INSPECTOR", func() {
		if data.InspectorName != "" {
			g.addTextLine(doc, data.InspectorName, false)
		}
		if data.InspectorCompany != "" {
			g.addTextLine(doc, data.InspectorCompany, false)
		}
		if data.InspectorLicense != "" {
			g.addTextLine(doc, "License: "+data.InspectorLicense, false)
		}
	})

	// Client information
	if data.HasClient() {
		g.addLabeledSection(doc, "CLIENT", func() {
			g.addTextLine(doc, data.ClientName, false)
			if data.ClientEmail != "" {
				g.addTextLine(doc, data.ClientEmail, false)
			}
			if data.ClientPhone != "" {
				g.addTextLine(doc, data.ClientPhone, false)
			}
		})
	}

	// Page break
	doc.AddParagraph().AddRun().AddPageBreak()
}

// =============================================================================
// Executive Summary
// =============================================================================

func (g *DOCXGenerator) addExecutiveSummary(doc *document.Document, data *domain.ReportData) {
	g.addSectionHeader(doc, "Executive Summary")

	// Violation counts
	g.addSubsectionHeader(doc, "Violations Summary")

	counts := data.ViolationCountBySeverity()
	total := data.TotalViolations()

	// Create summary table
	table := doc.AddTable()
	table.Properties().SetWidthPercent(60)

	// Header row
	headerRow := table.AddRow()
	g.addTableCell(headerRow, "Severity", true, "")
	g.addTableCell(headerRow, "Count", true, "")

	// Data rows
	severities := []domain.ViolationSeverity{
		domain.ViolationSeverityCritical,
		domain.ViolationSeveritySerious,
		domain.ViolationSeverityOther,
		domain.ViolationSeverityRecommendation,
	}

	for _, sev := range severities {
		count := counts[sev]
		if count > 0 || sev == domain.ViolationSeverityCritical || sev == domain.ViolationSeveritySerious {
			row := table.AddRow()
			g.addTableCell(row, SeverityLabel(sev), false, SeverityColor(sev))
			g.addTableCell(row, fmt.Sprintf("%d", count), false, "")
		}
	}

	// Total row
	totalRow := table.AddRow()
	g.addTableCell(totalRow, "Total", true, "")
	g.addTableCell(totalRow, fmt.Sprintf("%d", total), true, "")

	doc.AddParagraph() // Spacing

	// Conditions
	if data.WeatherConditions != "" || data.Temperature != "" {
		g.addSubsectionHeader(doc, "Conditions During Inspection")
		if data.WeatherConditions != "" {
			g.addTextLine(doc, "Weather: "+data.WeatherConditions, false)
		}
		if data.Temperature != "" {
			g.addTextLine(doc, "Temperature: "+data.Temperature, false)
		}
	}

	// Inspector notes
	if data.InspectorNotes != "" {
		g.addSubsectionHeader(doc, "General Notes")
		g.addTextLine(doc, data.InspectorNotes, false)
	}

	doc.AddParagraph().AddRun().AddPageBreak()
}

// =============================================================================
// Site Information
// =============================================================================

func (g *DOCXGenerator) addSiteInformation(doc *document.Document, data *domain.ReportData) {
	if !data.HasSite() && !data.HasClient() {
		return
	}

	g.addSectionHeader(doc, "Site & Client Information")

	if data.HasSite() {
		g.addSubsectionHeader(doc, "Site Details")
		g.addLabelValue(doc, "Site Name", data.SiteName)
		g.addLabelValue(doc, "Address", data.SiteFullAddress())
	}

	if data.HasClient() {
		g.addSubsectionHeader(doc, "Client Details")
		g.addLabelValue(doc, "Client Name", data.ClientName)
		if data.ClientEmail != "" {
			g.addLabelValue(doc, "Email", data.ClientEmail)
		}
		if data.ClientPhone != "" {
			g.addLabelValue(doc, "Phone", data.ClientPhone)
		}
	}

	g.addSubsectionHeader(doc, "Inspector Details")
	g.addLabelValue(doc, "Name", data.InspectorName)
	if data.InspectorCompany != "" {
		g.addLabelValue(doc, "Company", data.InspectorCompany)
	}
	if data.InspectorLicense != "" {
		g.addLabelValue(doc, "License #", data.InspectorLicense)
	}
	if data.InspectorEmail != "" {
		g.addLabelValue(doc, "Email", data.InspectorEmail)
	}
	if data.InspectorPhone != "" {
		g.addLabelValue(doc, "Phone", data.InspectorPhone)
	}

	doc.AddParagraph().AddRun().AddPageBreak()
}

// =============================================================================
// Findings Section
// =============================================================================

func (g *DOCXGenerator) addFindings(doc *document.Document, data *domain.ReportData) {
	g.addSectionHeader(doc, "Inspection Findings")

	if len(data.Violations) == 0 {
		para := doc.AddParagraph()
		run := para.AddRun()
		run.Properties().SetItalic(true)
		run.AddText("No violations were identified during this inspection.")
		return
	}

	for i, violation := range data.Violations {
		g.addViolation(doc, violation, i+1)

		if i < len(data.Violations)-1 {
			// Add separator between violations (using spacing)
			sep := doc.AddParagraph()
			sep.Properties().SetSpacing(10*measurement.Point, 10*measurement.Point)
			sepRun := sep.AddRun()
			sepRun.Properties().SetColor(color.LightGray)
			sepRun.AddText("────────────────────────────────────────")
		}
	}

	doc.AddParagraph().AddRun().AddPageBreak()
}

func (g *DOCXGenerator) addViolation(doc *document.Document, violation domain.ReportViolation, number int) {
	// Violation header
	header := doc.AddParagraph()
	headerRun := header.AddRun()
	headerRun.Properties().SetBold(true)
	headerRun.Properties().SetSize(14 * measurement.Point)
	headerRun.AddText(fmt.Sprintf("Finding #%d", number))

	// Severity
	severity := doc.AddParagraph()
	sevLabel := severity.AddRun()
	sevLabel.AddText("Severity: ")
	sevValue := severity.AddRun()
	sevValue.Properties().SetBold(true)
	r, g_, b := HexToRGB(SeverityColor(violation.Severity))
	sevValue.Properties().SetColor(color.RGB(uint8(r), uint8(g_), uint8(b)))
	sevValue.AddText(SeverityLabel(violation.Severity))

	// Description
	descLabel := doc.AddParagraph()
	descLabelRun := descLabel.AddRun()
	descLabelRun.Properties().SetBold(true)
	descLabelRun.AddText("Description:")

	descValue := doc.AddParagraph()
	descValue.AddRun().AddText(violation.Description)

	// Primary regulation
	if reg := violation.PrimaryRegulation(); reg != nil {
		regLabel := doc.AddParagraph()
		regLabelRun := regLabel.AddRun()
		regLabelRun.Properties().SetBold(true)
		regLabelRun.AddText("OSHA Regulation:")

		regStd := doc.AddParagraph()
		regStdRun := regStd.AddRun()
		regStdRun.Properties().SetBold(true)
		regStdRun.Properties().SetColor(color.RGB(255, 107, 53)) // Safety orange
		regStdRun.AddText(reg.StandardNumber)

		if reg.Title != "" {
			regTitle := doc.AddParagraph()
			regTitle.AddRun().AddText(reg.Title)
		}

		if reg.Category != "" {
			regCat := doc.AddParagraph()
			catRun := regCat.AddRun()
			catRun.Properties().SetColor(color.Gray)
			catRun.AddText("Category: " + reg.Category)
		}
	}

	// Inspector notes
	if violation.InspectorNotes != "" {
		notesLabel := doc.AddParagraph()
		notesLabelRun := notesLabel.AddRun()
		notesLabelRun.Properties().SetBold(true)
		notesLabelRun.AddText("Inspector Notes:")

		notesValue := doc.AddParagraph()
		notesRun := notesValue.AddRun()
		notesRun.Properties().SetItalic(true)
		notesRun.AddText(violation.InspectorNotes)
	}

	doc.AddParagraph() // Spacing
}

// =============================================================================
// Appendix
// =============================================================================

func (g *DOCXGenerator) addAppendix(doc *document.Document, data *domain.ReportData) {
	g.addSectionHeader(doc, "Appendix: Regulation Reference")

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
		para := doc.AddParagraph()
		run := para.AddRun()
		run.Properties().SetItalic(true)
		run.AddText("No regulations cited in this report.")
		return
	}

	for stdNum, reg := range regulations {
		// Standard number
		stdPara := doc.AddParagraph()
		stdRun := stdPara.AddRun()
		stdRun.Properties().SetBold(true)
		stdRun.Properties().SetSize(12 * measurement.Point)
		stdRun.AddText(stdNum)

		// Title
		if reg.Title != "" {
			titlePara := doc.AddParagraph()
			titleRun := titlePara.AddRun()
			titleRun.Properties().SetBold(true)
			titleRun.AddText(reg.Title)
		}

		// Category
		if reg.Category != "" {
			catPara := doc.AddParagraph()
			catRun := catPara.AddRun()
			catRun.Properties().SetItalic(true)
			catRun.Properties().SetColor(color.Gray)
			catRun.AddText("Category: " + reg.Category)
		}

		// Full text
		if reg.FullText != "" {
			fullText := reg.FullText
			if len(fullText) > 1000 {
				fullText = fullText[:1000] + "..."
			}
			textPara := doc.AddParagraph()
			textRun := textPara.AddRun()
			textRun.Properties().SetSize(9 * measurement.Point)
			textRun.AddText(fullText)
		}

		// Separator (using spacing and text)
		sep := doc.AddParagraph()
		sep.Properties().SetSpacing(8*measurement.Point, 8*measurement.Point)
		sepRun := sep.AddRun()
		sepRun.Properties().SetColor(color.LightGray)
		sepRun.AddText("────────────────────────────────────────")
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

func (g *DOCXGenerator) addSectionHeader(doc *document.Document, title string) {
	para := doc.AddParagraph()
	run := para.AddRun()
	run.Properties().SetBold(true)
	run.Properties().SetSize(18 * measurement.Point)
	run.Properties().SetColor(color.RGB(30, 58, 95)) // Navy
	run.AddText(title)
	para.Properties().SetSpacing(0, 12*measurement.Point)

	// Add underline effect with a second paragraph
	underline := doc.AddParagraph()
	underlineRun := underline.AddRun()
	underlineRun.Properties().SetColor(color.RGB(30, 58, 95))
	underlineRun.AddText("══════════════════════════════════════════════════")
	underline.Properties().SetSpacing(0, 12*measurement.Point)
}

func (g *DOCXGenerator) addSubsectionHeader(doc *document.Document, title string) {
	para := doc.AddParagraph()
	run := para.AddRun()
	run.Properties().SetBold(true)
	run.Properties().SetSize(12 * measurement.Point)
	run.AddText(title)
	para.Properties().SetSpacing(12*measurement.Point, 6*measurement.Point)
}

func (g *DOCXGenerator) addLabeledSection(doc *document.Document, label string, content func()) {
	// Label
	labelPara := doc.AddParagraph()
	labelRun := labelPara.AddRun()
	labelRun.Properties().SetBold(true)
	labelRun.Properties().SetSize(10 * measurement.Point)
	labelRun.Properties().SetColor(color.Gray)
	labelRun.AddText(label)
	labelPara.Properties().SetSpacing(12*measurement.Point, 4*measurement.Point)

	// Content
	content()
}

func (g *DOCXGenerator) addTextLine(doc *document.Document, text string, italic bool) {
	para := doc.AddParagraph()
	run := para.AddRun()
	if italic {
		run.Properties().SetItalic(true)
	}
	run.AddText(text)
}

func (g *DOCXGenerator) addLabelValue(doc *document.Document, label, value string) {
	if value == "" {
		return
	}
	para := doc.AddParagraph()
	labelRun := para.AddRun()
	labelRun.Properties().SetBold(true)
	labelRun.AddText(label + ": ")
	para.AddRun().AddText(value)
}

func (g *DOCXGenerator) addTableCell(row document.Row, text string, bold bool, colorHex string) {
	cell := row.AddCell()
	para := cell.AddParagraph()
	run := para.AddRun()
	if bold {
		run.Properties().SetBold(true)
	}
	if colorHex != "" {
		r, g_, b := HexToRGB(colorHex)
		run.Properties().SetColor(color.RGB(uint8(r), uint8(g_), uint8(b)))
	}
	run.AddText(text)
}

func (g *DOCXGenerator) hasRegulations(data *domain.ReportData) bool {
	for _, v := range data.Violations {
		if len(v.Regulations) > 0 {
			return true
		}
	}
	return false
}
