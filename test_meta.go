package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sessiondb/internal/repository"
)

func main() {
	dsn := "host=localhost user=sessiondb password=sessiondb dbname=sessiondb port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	metaRepo := repository.NewMetadataRepository(db)

	instanceID, _ := uuid.Parse("11258544-6d66-4ba9-8329-11891d9baf2d")
	member := "'mouli'@'%'"

	roles, err := metaRepo.FindRoleMembershipsByMember(instanceID, member)
	fmt.Printf("Roles for %s: %v (err: %v)\n", member, roles, err)
}
