package extractors

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExcelExtractor extracts text from Excel files (.xlsx, .xls)
type ExcelExtractor struct{}

// NewExcelExtractor creates a new Excel extractor
func NewExcelExtractor() *ExcelExtractor {
	return &ExcelExtractor{}
}

// CanExtract returns true for Excel MIME types
func (e *ExcelExtractor) CanExtract(mimeType string) bool {
	excelTypes := []string{
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", // .xlsx
		"application/vnd.ms-excel",                                          // .xls
		"application/vnd.ms-excel.sheet.macroenabled.12",                    // .xlsm
	}

	for _, t := range excelTypes {
		if strings.HasPrefix(mimeType, t) || mimeType == t {
			return true
		}
	}

	return false
}

// Extract extracts text content from Excel data
func (e *ExcelExtractor) Extract(ctx context.Context, data []byte) (string, error) {
	// Open Excel file from bytes
	reader := bytes.NewReader(data)
	workbook, err := excelize.OpenReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer workbook.Close()

	var content strings.Builder
	sheetList := workbook.GetSheetList()

	if len(sheetList) == 0 {
		return "", fmt.Errorf("no sheets found in Excel file")
	}

	hasContent := false

	// Extract each sheet
	for sheetIdx, sheetName := range sheetList {
		if sheetIdx > 0 {
			content.WriteString("\n\n")
		}

		content.WriteString(fmt.Sprintf("Sheet: %s\n", sheetName))
		content.WriteString(strings.Repeat("-", len(sheetName)+7))
		content.WriteString("\n\n")

		rows, err := workbook.GetRows(sheetName)
		if err != nil {
			content.WriteString(fmt.Sprintf("Error reading sheet: %v\n", err))
			continue
		}

		if len(rows) == 0 {
			content.WriteString("(Empty sheet)\n")
			continue
		}

		// Limit rows to prevent token overflow
		maxRows := 1000
		if len(rows) > maxRows {
			rows = rows[:maxRows]
			content.WriteString(fmt.Sprintf("(Showing first %d rows of %d)\n\n", maxRows, len(rows)))
		}

		// Find max columns across all rows
		maxCols := 0
		for _, row := range rows {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}

		if maxCols == 0 {
			content.WriteString("(No data)\n")
			continue
		}

		// Format as markdown table
		for i, row := range rows {
			// Skip completely empty rows
			if len(strings.TrimSpace(strings.Join(row, ""))) == 0 {
				continue
			}

			// Pad row to max columns for alignment
			paddedRow := make([]string, maxCols)
			for j := 0; j < maxCols; j++ {
				if j < len(row) {
					paddedRow[j] = row[j]
				} else {
					paddedRow[j] = ""
				}
			}

			// Join cells with pipe separators (markdown table)
			content.WriteString("| ")
			content.WriteString(strings.Join(paddedRow, " | "))
			content.WriteString(" |\n")

			// Add header separator after first row
			if i == 0 && len(rows) > 1 {
				content.WriteString("|")
				for j := 0; j < maxCols; j++ {
					content.WriteString(" --- |")
				}
				content.WriteString("\n")
			}

			hasContent = true
		}
	}

	if !hasContent {
		return "", fmt.Errorf("no content extracted from Excel file")
	}

	return content.String(), nil
}
