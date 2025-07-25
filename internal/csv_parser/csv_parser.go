package csvParser

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"github.com/abhinavxd/libredesk/internal/envelope"
)	

// ParseCSV reads a CSV file from the provided reader and returns a slice of string slices,
// where each inner slice represents a row of the CSV file. It returns an error if the CSV
// cannot be parsed.

func ParseCSV(r io.Reader) ([]map[string]string, error) {
	csvReader := csv.NewReader(r)
	csvReader.Comma = ',' // Set the delimiter to comma
	csvReader.TrimLeadingSpace = true

	// Read all records from the CSV
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, envelope.NewError(envelope.InputError, "Failed to parse CSV file", err)
	}

	if len(records) == 0 {
		return nil, errors.New("CSV file is empty")
	}

	// Check for empty rows
	for _, record := range records {
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			return nil, envelope.NewError(envelope.InputError, "CSV file contains empty rows", nil)
		}
	}
	// Convert records to a map with headers as keys
	headers := records[0]
	recordsMap := make([]map[string]string, len(records)-1)

	for i, row := range records[1:] {
		if len(row) != len(headers) {
			return nil, envelope.NewError(envelope.InputError, "Row length does not match header length", nil)
		}
		recordMap := make(map[string]string)
		for j, value := range row {
			recordMap[headers[j]] = value
		}
		recordsMap[i] = recordMap
	}

	return recordsMap, nil
}
