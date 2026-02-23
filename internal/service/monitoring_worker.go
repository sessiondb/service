package service

import (
	"log"
	"time"
)

type MonitoringWorker struct {
	MonitoringService *MonitoringService
}

func NewMonitoringWorker(monitoringService *MonitoringService) *MonitoringWorker {
	return &MonitoringWorker{
		MonitoringService: monitoringService,
	}
}

func (w *MonitoringWorker) Start() {
	log.Println("Monitoring Worker started")

	// Run every 10 minutes as requested
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		// Run immediately once on start
		w.runChecks()

		for range ticker.C {
			w.runChecks()
		}
	}()
}

func (w *MonitoringWorker) runChecks() {
	instances, err := w.MonitoringService.InstanceRepo.FindAll()
	if err != nil {
		log.Printf("Monitoring Worker Error fetching instances: %v", err)
		return
	}

	for _, inst := range instances {
		if inst.MonitoringEnabled {
			log.Printf("Monitoring Worker: Checking %s", inst.Name)
			err := w.MonitoringService.MonitorInstance(inst.ID)
			if err != nil {
				log.Printf("Monitoring Worker: Error monitoring %s: %v", inst.Name, err)
			}
		}
	}
}
