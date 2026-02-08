package service

import (
	"database/sql"
	"fmt"
	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // Postgres driver
)

type QueryService struct {
	QueryRepo *repository.QueryRepository
	Config    *config.Config
}

func NewQueryService(queryRepo *repository.QueryRepository, cfg *config.Config) *QueryService {
	return &QueryService{
		QueryRepo: queryRepo,
		Config:    cfg,
	}
}

func (s *QueryService) ExecuteQuery(userID uuid.UUID, database, query string) (interface{}, error) {
	// TODO: Check permissions here using a PermissionService or Auth middleware context

	// Use default database from config if not provided
	dbName := database
	if dbName == "" {
		dbName = s.Config.Database.Name
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		s.Config.Database.Host,
		s.Config.Database.User,
		s.Config.Database.Password,
		dbName,
		s.Config.Database.Port,
		s.Config.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	start := time.Now()
	rows, err := db.Query(query)
	duration := time.Since(start).Milliseconds()

	status := "success"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	// Async log history
	go func() {
		history := &models.QueryHistory{
			UserID:          userID,
			Query:           query,
			Database:        dbName,
			ExecutionTimeMs: duration,
			RowsReturned:    0, // Update below if successful
			Status:          status,
			ErrorMessage:    errMsg,
			ExecutedAt:      start,
		}
		s.QueryRepo.SaveHistory(history)
	}()

	if err != nil {
		// Return friendly message if it's not a SELECT query or just an error
		// Doc allows: { "message": "...", "rowsAffected": 1 }
		return map[string]interface{}{
			"message":      err.Error(),
			"rowsAffected": 0,
		}, nil // Return as valid response or error? Doc implies valid JSON response for non-SELECT usually. But if it's an error, maybe error?
		// Let's return error for now if it failed.
		return nil, err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	// Result format: { columns: [], rows: [[...], [...]], rowCount: N }
	var dataRows [][]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		rows.Scan(valuePtrs...)

		row := make([]interface{}, len(columns))
		for i := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				row[i] = string(b)
			} else {
				row[i] = val
			}
		}
		dataRows = append(dataRows, row)
	}

	if dataRows == nil {
		dataRows = [][]interface{}{}
	}

	return map[string]interface{}{
		"columns":  columns,
		"rows":     dataRows,
		"rowCount": len(dataRows),
	}, nil
}

func (s *QueryService) GetHistory(userID uuid.UUID) ([]models.QueryHistory, error) {
	return s.QueryRepo.GetHistory(userID)
}

func (s *QueryService) SaveScript(userID uuid.UUID, name, query string, isPublic bool) (*models.SavedScript, error) {
	script := &models.SavedScript{
		UserID:   userID,
		Name:     name,
		Query:    query,
		IsPublic: isPublic,
	}
	if err := s.QueryRepo.SaveScript(script); err != nil {
		return nil, err
	}
	return script, nil
}

func (s *QueryService) GetScripts(userID uuid.UUID) ([]models.SavedScript, error) {
	return s.QueryRepo.GetScripts(userID)
}
