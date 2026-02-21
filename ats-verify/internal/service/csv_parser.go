package service

import (
	"bytes"
	"encoding/csv"
	"io"
)

// NewRobustCSVReader creates a CSV reader that handles BOM, detects ';' vs ',',
// and sets LazyQuotes to handle malformed data.
func NewRobustCSVReader(reader io.Reader) (*csv.Reader, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// 1. Remove UTF-8 BOM if present
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	// 2. Detect separator: look at the first line
	firstLineEnd := bytes.IndexByte(data, '\n')
	var firstLine []byte
	if firstLineEnd == -1 {
		firstLine = data
	} else {
		firstLine = data[:firstLineEnd]
	}

	comma := ','
	testReader := csv.NewReader(bytes.NewReader(firstLine))
	testReader.Comma = ','
	testReader.LazyQuotes = true
	testRecord, err := testReader.Read()

	if err == nil && len(testRecord) == 1 && bytes.Contains(firstLine, []byte(";")) {
		comma = ';'
	}

	// 3. Create reader with robust settings
	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = comma
	csvReader.TrimLeadingSpace = true // Trims leading space of field
	csvReader.LazyQuotes = true       // Allow unescaped quotes
	csvReader.FieldsPerRecord = -1    // Allow variable number of fields

	return csvReader, nil
}
