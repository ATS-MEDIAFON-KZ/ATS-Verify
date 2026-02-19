package service

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ledongthuc/pdf"
)

// PDFExtractor extracts plain text from PDF files.
// Primary implementation uses ledongthuc/pdf.
// If extraction quality is insufficient for complex layouts (Graph 31),
// fallback to Python PyMuPDF sidecar (see findings.md).
type PDFExtractor struct{}

// NewPDFExtractor creates a new PDFExtractor.
func NewPDFExtractor() *PDFExtractor {
	return &PDFExtractor{}
}

// ExtractTextFromFile extracts all text from a PDF file on disk.
func (e *PDFExtractor) ExtractTextFromFile(filePath string) (string, error) {
	f, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening PDF %s: %w", filePath, err)
	}
	defer f.Close()

	return extractText(reader)
}

// ExtractTextFromReader extracts text from PDF bytes (e.g. multipart upload).
// ledongthuc/pdf requires a file path, so we write to a temp file first.
func (e *PDFExtractor) ExtractTextFromReader(r io.Reader) (string, error) {
	tmpFile, err := os.CreateTemp("", "ats-verify-pdf-*.pdf")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, r); err != nil {
		return "", fmt.Errorf("writing temp PDF: %w", err)
	}
	tmpFile.Close() // Close before reading.

	return e.ExtractTextFromFile(tmpFile.Name())
}

// extractText reads all plain text from a pdf.Reader.
func extractText(reader *pdf.Reader) (string, error) {
	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extracting plain text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(textReader); err != nil {
		return "", fmt.Errorf("reading text buffer: %w", err)
	}

	return buf.String(), nil
}
