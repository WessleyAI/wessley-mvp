package manuals

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// ExtractTextFromPDF does basic text extraction from a PDF file.
// This is a simple implementation that extracts text between stream/endstream
// markers and BT/ET text blocks. For production use, a proper PDF library
// would be needed.
func ExtractTextFromPDF(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read pdf: %w", err)
	}

	return extractPDFText(data), nil
}

// extractPDFText extracts readable text content from raw PDF bytes.
func extractPDFText(data []byte) string {
	var texts []string

	// Look for text between parentheses in BT...ET blocks
	// This is a simplified approach for basic PDF text extraction
	inText := false
	for i := 0; i < len(data)-1; i++ {
		// Detect BT (Begin Text)
		if data[i] == 'B' && data[i+1] == 'T' && (i == 0 || !isAlpha(data[i-1])) {
			inText = true
			continue
		}
		// Detect ET (End Text)
		if data[i] == 'E' && data[i+1] == 'T' && inText && (i+2 >= len(data) || !isAlpha(data[i+2])) {
			inText = false
			continue
		}

		if inText && data[i] == '(' {
			// Extract text between parentheses
			end := bytes.IndexByte(data[i+1:], ')')
			if end >= 0 {
				text := string(data[i+1 : i+1+end])
				text = cleanPDFText(text)
				if text != "" {
					texts = append(texts, text)
				}
				i += end + 1
			}
		}
	}

	return strings.Join(texts, " ")
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// cleanPDFText removes common PDF escape sequences.
func cleanPDFText(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\(", "(")
	s = strings.ReplaceAll(s, "\\)", ")")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return strings.TrimSpace(s)
}
