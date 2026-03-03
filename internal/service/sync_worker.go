// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"log"
	"time"
)

type SyncWorker struct {
	SyncService *SyncService
	Hub         *NotificationHub
}

func NewSyncWorker(syncService *SyncService, hub *NotificationHub) *SyncWorker {
	return &SyncWorker{
		SyncService: syncService,
		Hub:         hub,
	}
}

func (w *SyncWorker) Start() {
	log.Println("Sync Worker started")

	// 1. Listen for sync updates and broadcast to WS Hub
	go func() {
		for update := range w.SyncService.UpdateChan {
			log.Printf("Worker broadcasting sync update: %+v", update)
			w.Hub.Broadcast(update)
		}
	}()

	// 2. Periodic Auto-Sync (Initial implementation: check every 1 hor)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			w.triggerPeriodicSync()
		}
	}()
}

func (w *SyncWorker) triggerPeriodicSync() {
	instances, err := w.SyncService.InstanceRepo.FindAll()
	if err != nil {
		log.Printf("Worker Error fetching instances: %v", err)
		return
	}

	for _, inst := range instances {
		// Only sync if last sync was more than 1 hour ago or never
		if inst.LastSync == nil || time.Since(*inst.LastSync) > 1*time.Hour {
			log.Printf("Worker Triggering auto-sync for %s", inst.Name)
			go w.SyncService.SyncInstance(inst.ID, "")
		}
	}
}
