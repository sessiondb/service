package service

import (
	"database/sql"
	"fmt"
	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // Postgres driver
)

type QueryService struct {
	QueryRepo    *repository.QueryRepository
	InstanceRepo *repository.InstanceRepository
	Config       *config.Config
}

func NewQueryService(queryRepo *repository.QueryRepository, cfg *config.Config) *QueryService {
	return &QueryService{
		QueryRepo: queryRepo,
		Config:    cfg,
	}
}

// SetInstanceRepo allows injection of InstanceRepository after construction
func (s *QueryService) SetInstanceRepo(repo *repository.InstanceRepository) {
	s.InstanceRepo = repo
}

func (s *QueryService) ExecuteQuery(userID uuid.UUID, instanceID uuid.UUID, query string) (interface{}, error) {
	// Lookup instance
	instance, err := s.InstanceRepo.FindByID(instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Build DSN based on instance type
	var dsn string
	var driverName string
	switch instance.Type {
	case "postgres":
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
			instance.Host, instance.Port, instance.Username, instance.Password)
	case "mysql":
		driverName = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/",
			instance.Username, instance.Password, instance.Host, instance.Port)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", instance.Type)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
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
			Database:        instance.Name,
			ExecutionTimeMs: duration,
			RowsReturned:    0,
			Status:          status,
			ErrorMessage:    errMsg,
			ExecutedAt:      start,
		}
		s.QueryRepo.SaveHistory(history)
	}()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
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
