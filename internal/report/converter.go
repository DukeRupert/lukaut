package report

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DukeRupert/lukaut/internal/domain"
)

// Converter transforms HTML content to a specific output format.
type Converter interface {
	// Convert transforms HTML content and writes the result to w.
	Convert(ctx context.Context, html []byte, w io.Writer) error

	// Format returns the output format of this converter.
	Format() domain.ReportFormat
}

// =============================================================================
// WeasyPrint Converter (HTML → PDF)
// =============================================================================

// WeasyPrintConverter converts HTML to PDF using WeasyPrint.
// Requires weasyprint to be installed: pip install weasyprint
type WeasyPrintConverter struct {
	// Command is the weasyprint command to execute. Defaults to "weasyprint".
	Command string
}

// NewWeasyPrintConverter creates a new WeasyPrint converter.
func NewWeasyPrintConverter() *WeasyPrintConverter {
	return &WeasyPrintConverter{
		Command: "weasyprint",
	}
}

// Format returns the output format (PDF).
func (c *WeasyPrintConverter) Format() domain.ReportFormat {
	return domain.ReportFormatPDF
}

// Convert transforms HTML to PDF using WeasyPrint.
func (c *WeasyPrintConverter) Convert(ctx context.Context, html []byte, w io.Writer) error {
	// Create temp directory for input/output files
	tmpDir, err := os.MkdirTemp("", "report-pdf-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.html")
	outputPath := filepath.Join(tmpDir, "output.pdf")

	// Write HTML to temp file
	if err := os.WriteFile(inputPath, html, 0644); err != nil {
		return fmt.Errorf("write input file: %w", err)
	}

	// Execute weasyprint
	cmd := exec.CommandContext(ctx, c.Command, inputPath, outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("weasyprint failed: %w, stderr: %s", err, stderr.String())
	}

	// Read output file
	pdfData, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("read output file: %w", err)
	}

	// Write to output writer
	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// =============================================================================
// Pandoc Converter (HTML → DOCX)
// =============================================================================

// PandocConverter converts HTML to DOCX using Pandoc.
// Requires pandoc to be installed: apt-get install pandoc
type PandocConverter struct {
	// Command is the pandoc command to execute. Defaults to "pandoc".
	Command string

	// ReferenceDoc is an optional path to a reference.docx for styling.
	// If empty, Pandoc's default styling is used.
	ReferenceDoc string
}

// NewPandocConverter creates a new Pandoc converter.
func NewPandocConverter() *PandocConverter {
	return &PandocConverter{
		Command: "pandoc",
	}
}

// Format returns the output format (DOCX).
func (c *PandocConverter) Format() domain.ReportFormat {
	return domain.ReportFormatDOCX
}

// Convert transforms HTML to DOCX using Pandoc.
func (c *PandocConverter) Convert(ctx context.Context, html []byte, w io.Writer) error {
	// Create temp directory for input/output files
	tmpDir, err := os.MkdirTemp("", "report-docx-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.html")
	outputPath := filepath.Join(tmpDir, "output.docx")

	// Write HTML to temp file
	if err := os.WriteFile(inputPath, html, 0644); err != nil {
		return fmt.Errorf("write input file: %w", err)
	}

	// Build pandoc command
	args := []string{
		inputPath,
		"-o", outputPath,
		"--from=html",
		"--to=docx",
	}

	// Add reference doc if specified
	if c.ReferenceDoc != "" {
		args = append(args, "--reference-doc="+c.ReferenceDoc)
	}

	// Execute pandoc
	cmd := exec.CommandContext(ctx, c.Command, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc failed: %w, stderr: %s", err, stderr.String())
	}

	// Read output file
	docxData, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("read output file: %w", err)
	}

	// Write to output writer
	if _, err := w.Write(docxData); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// =============================================================================
// Converter Availability Checks
// =============================================================================

// IsWeasyPrintAvailable checks if weasyprint is installed and accessible.
func IsWeasyPrintAvailable() bool {
	_, err := exec.LookPath("weasyprint")
	return err == nil
}

// IsPandocAvailable checks if pandoc is installed and accessible.
func IsPandocAvailable() bool {
	_, err := exec.LookPath("pandoc")
	return err == nil
}
