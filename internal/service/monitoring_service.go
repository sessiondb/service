package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
)

type MonitoringService struct {
	InstanceRepo *repository.InstanceRepository
	MonitorRepo  *repository.MonitoringRepository
	Hub          *NotificationHub
}

func NewMonitoringService(instanceRepo *repository.InstanceRepository, monitorRepo *repository.MonitoringRepository, hub *NotificationHub) *MonitoringService {
	return &MonitoringService{
		InstanceRepo: instanceRepo,
		MonitorRepo:  monitorRepo,
		Hub:          hub,
	}
}

type MySQLMetrics struct {
	Uptime       int64             `json:"uptime"`
	Threads      int               `json:"threads_connected"`
	Queries      int64             `json:"queries"`
	GlobalStatus map[string]string `json:"global_status,omitempty"`
}

func (s *MonitoringService) MonitorInstance(instanceID uuid.UUID) error {
	instance, err := s.InstanceRepo.FindByID(instanceID)
	if err != nil {
		return err
	}

	if !instance.MonitoringEnabled {
		return nil
	}

	status := "online"
	var metrics MySQLMetrics
	message := "Checked successfully"

	// Connect and check
	dsn := s.getDSN(instance)
	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		status = "offline"
		message = fmt.Sprintf("Failed to connect: %v", err)
	} else {
		defer db.Close()
		err = db.Ping()
		if err != nil {
			status = "offline"
			message = fmt.Sprintf("Ping failed: %v", err)
		} else {
			// Fetch MySQL metrics if type is mysql
			if instance.Type == "mysql" {
				metrics, err = s.fetchMySQLMetrics(db)
				if err != nil {
					log.Printf("Error fetching metrics for %s: %v", instance.Name, err)
				}
			}
		}
	}

	// Update instance status if it changed
	if instance.Status != status {
		oldStatus := instance.Status
		instance.Status = status
		s.InstanceRepo.Update(instance)

		// Alerting logic
		if instance.IsProd && status == "offline" {
			s.triggerAlert(instance, "Instance is DOWN", fmt.Sprintf("Production instance %s is offline. Error: %s", instance.Name, message))
		} else if status == "online" && oldStatus == "offline" {
			s.triggerAlert(instance, "Instance is UP", fmt.Sprintf("Instance %s is back online.", instance.Name))
		}
	}

	// Save Log
	metricsJSON, _ := json.Marshal(metrics)
	monLog := &models.DBMonitoringLog{
		InstanceID: instanceID,
		Status:     status,
		Uptime:     metrics.Uptime,
		Metrics:    metricsJSON,
		Message:    message,
	}

	return s.MonitorRepo.CreateLog(monLog)
}

func (s *MonitoringService) fetchMySQLMetrics(db *sql.DB) (MySQLMetrics, error) {
	var m MySQLMetrics
	m.GlobalStatus = make(map[string]string)

	rows, err := db.Query("SHOW GLOBAL STATUS WHERE Variable_name IN ('Uptime', 'Threads_connected', 'Queries')")
	if err != nil {
		return m, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		m.GlobalStatus[name] = value
		switch name {
		case "Uptime":
			fmt.Sscanf(value, "%d", &m.Uptime)
		case "Threads_connected":
			fmt.Sscanf(value, "%d", &m.Threads)
		case "Queries":
			fmt.Sscanf(value, "%d", &m.Queries)
		}
	}

	return m, nil
}

func (s *MonitoringService) triggerAlert(instance *models.DBInstance, title, message string) {
	log.Printf("ALERT [%s]: %s - %s", instance.Name, title, message)

	// 1. Broadcast to WS
	alert := map[string]interface{}{
		"type":       "MONITORING_ALERT",
		"instanceId": instance.ID,
		"name":       instance.Name,
		"title":      title,
		"message":    message,
		"severity":   "critical",
		"isProd":     instance.IsProd,
		"timestamp":  time.Now(),
	}
	s.Hub.Broadcast(alert)

	// 2. Mock Email
	if instance.AlertEmail != "" {
		log.Printf("SENDING EMAIL to %s: Subject: %s, Body: %s", instance.AlertEmail, title, message)
	}
}

func (s *MonitoringService) getDSN(instance *models.DBInstance) string {
	switch instance.Type {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true",
			instance.Username, instance.Password, instance.Host, instance.Port)
	case "postgres":
		return fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=disable",
			instance.Host, instance.Username, instance.Password, instance.Port)
	}
	return ""
}
