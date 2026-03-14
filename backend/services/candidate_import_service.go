package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"ai-recruiter/backend/models"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CandidateImportService struct {
	db *mongo.Client
}

func NewCandidateImportService(db *mongo.Client) *CandidateImportService {
	return &CandidateImportService{db: db}
}

// ParseCSV parses CSV content and returns candidate records
func (s *CandidateImportService) ParseCSV(data []byte) ([]models.Candidate, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map header columns
	columnMap := s.mapColumns(header)

	var candidates []models.Candidate

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %w", err)
		}

		candidate := s.parseRecord(record, columnMap)
		if candidate.Name != "" || candidate.Email != "" {
			candidates = append(candidates, candidate)
		}
	}

	return candidates, nil
}

// ParseExcel parses Excel content and returns candidate records
func (s *CandidateImportService) ParseExcel(data []byte) ([]models.Candidate, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Excel file: %w", err)
	}
	defer f.Close()

	// Get first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel file has no data rows")
	}

	// Map header columns
	columnMap := s.mapColumns(rows[0])

	var candidates []models.Candidate

	for i := 1; i < len(rows); i++ {
		candidate := s.parseRecord(rows[i], columnMap)
		if candidate.Name != "" || candidate.Email != "" {
			candidates = append(candidates, candidate)
		}
	}

	return candidates, nil
}

// ImportCandidates saves parsed candidates to MongoDB
func (s *CandidateImportService) ImportCandidates(candidates []models.Candidate) (int64, error) {
	if len(candidates) == 0 {
		return 0, fmt.Errorf("no candidates to import")
	}

	coll := s.db.Database("ai_recruiter").Collection("candidates")

	var docs []interface{}
	for i := range candidates {
		candidates[i].ID = primitive.NewObjectID()
		candidates[i].Status = "pending_screening"
		candidates[i].CreatedAt = time.Now()
		docs = append(docs, candidates[i])
	}

	result, err := coll.InsertMany(nil, docs)
	if err != nil {
		return 0, fmt.Errorf("failed to insert candidates: %w", err)
	}

	log.Printf("Imported %d candidates", len(result.InsertedIDs))
	return int64(len(result.InsertedIDs)), nil
}

// GetCandidates retrieves all candidates
func (s *CandidateImportService) GetCandidates() ([]models.Candidate, error) {
	coll := s.db.Database("ai_recruiter").Collection("candidates")

	cursor, err := coll.Find(nil, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query candidates: %w", err)
	}
	defer cursor.Close(nil)

	var candidates []models.Candidate
	if err = cursor.All(nil, &candidates); err != nil {
		return nil, fmt.Errorf("failed to decode candidates: %w", err)
	}

	return candidates, nil
}

// mapColumns creates a map of column names to indices
func (s *CandidateImportService) mapColumns(header []string) map[string]int {
	columnMap := make(map[string]int)
	for i, col := range header {
		normalized := strings.ToLower(strings.TrimSpace(col))
		columnMap[normalized] = i
	}
	return columnMap
}

// parseRecord converts a CSV/Excel record to a Candidate using the column map
func (s *CandidateImportService) parseRecord(record []string, columnMap map[string]int) models.Candidate {
	getField := func(key string) string {
		if idx, ok := columnMap[key]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	return models.Candidate{
		Name:   getField("name"),
		Email:  getField("email"),
		Phone:  getField("phone"),
		Role:   getField("role"),
		GitHub: getField("github"),
		LinkedIn: getField("linkedin"),
		Resume: getField("resume"),
	}
}
