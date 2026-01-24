package report

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"

	"github.com/DukeRupert/lukaut/internal/domain"
	reporttempl "github.com/DukeRupert/lukaut/internal/templ/report"
)

// =============================================================================
// HTML Generator
// =============================================================================

// HTMLGenerator generates reports by rendering an HTML template and converting
// to the target format using external tools (WeasyPrint for PDF, Pandoc for DOCX).
type HTMLGenerator struct {
	format        domain.ReportFormat
	pdfConverter  Converter
	docxConverter Converter
	logger        *slog.Logger
}

// NewHTMLGenerator creates a new HTML-based report generator.
func NewHTMLGenerator(format domain.ReportFormat, logger *slog.Logger) *HTMLGenerator {
	if logger == nil {
		logger = slog.Default()
	}
	return &HTMLGenerator{
		format:        format,
		pdfConverter:  NewWeasyPrintConverter(),
		docxConverter: NewPandocConverter(),
		logger:        logger,
	}
}

// Format returns the output format of this generator.
func (g *HTMLGenerator) Format() domain.ReportFormat {
	return g.format
}

// Generate creates a report and writes it to the provided writer.
func (g *HTMLGenerator) Generate(ctx context.Context, data *domain.ReportData, w io.Writer) (int64, error) {
	// 1. Prepare template data with image preprocessing
	templateData, err := g.prepareTemplateData(ctx, data)
	if err != nil {
		return 0, fmt.Errorf("prepare template data: %w", err)
	}

	// 2. Render HTML template
	var htmlBuf bytes.Buffer
	if err := reporttempl.Report(templateData).Render(ctx, &htmlBuf); err != nil {
		return 0, fmt.Errorf("render template: %w", err)
	}

	g.logger.Debug("HTML template rendered",
		"format", g.format,
		"html_size", htmlBuf.Len(),
		"violation_count", len(data.Violations),
	)

	// 3. Select converter based on format
	var converter Converter
	switch g.format {
	case domain.ReportFormatPDF:
		converter = g.pdfConverter
	case domain.ReportFormatDOCX:
		converter = g.docxConverter
	default:
		return 0, fmt.Errorf("unsupported format: %s", g.format)
	}

	// 4. Convert HTML to target format
	var outBuf bytes.Buffer
	if err := converter.Convert(ctx, htmlBuf.Bytes(), &outBuf); err != nil {
		return 0, fmt.Errorf("convert to %s: %w", g.format, err)
	}

	// 5. Write output
	n, err := w.Write(outBuf.Bytes())
	if err != nil {
		return int64(n), fmt.Errorf("write output: %w", err)
	}

	g.logger.Info("Report generated",
		"format", g.format,
		"size_bytes", n,
		"violation_count", len(data.Violations),
	)

	return int64(n), nil
}

// prepareTemplateData converts domain.ReportData to reporttempl.ReportTemplateData,
// optionally downloading and embedding images as base64 for DOCX generation.
func (g *HTMLGenerator) prepareTemplateData(ctx context.Context, data *domain.ReportData) (*reporttempl.ReportTemplateData, error) {
	templateData := &reporttempl.ReportTemplateData{
		ReportData: data,
	}

	// For DOCX, we need to embed images as base64 because Pandoc can't fetch remote URLs
	// For PDF, WeasyPrint can fetch URLs directly, so we skip this step
	if g.format == domain.ReportFormatDOCX {
		templateData.ImageDataMap = make(map[int]string)

		for _, v := range data.Violations {
			if v.ThumbnailURL == "" {
				continue
			}

			imgData, err := DownloadImage(ctx, v.ThumbnailURL)
			if err != nil {
				g.logger.Warn("Failed to download image for DOCX embedding",
					"violation_number", v.Number,
					"url", v.ThumbnailURL,
					"error", err,
				)
				continue
			}

			if imgData != nil {
				// Convert to base64 data URI
				dataURI := fmt.Sprintf("data:%s;base64,%s",
					imgData.ContentType,
					base64.StdEncoding.EncodeToString(imgData.Data),
				)
				templateData.ImageDataMap[v.Number] = dataURI
			}
		}

		g.logger.Debug("Images embedded for DOCX",
			"image_count", len(templateData.ImageDataMap),
		)
	}

	return templateData, nil
}

// =============================================================================
// Factory Functions for Backward Compatibility
// =============================================================================

// NewHTMLPDFGenerator creates an HTML generator for PDF output.
func NewHTMLPDFGenerator(logger *slog.Logger) *HTMLGenerator {
	return NewHTMLGenerator(domain.ReportFormatPDF, logger)
}

// NewHTMLDOCXGenerator creates an HTML generator for DOCX output.
func NewHTMLDOCXGenerator(logger *slog.Logger) *HTMLGenerator {
	return NewHTMLGenerator(domain.ReportFormatDOCX, logger)
}
