package repository

import (
	"fmt"
	"log"
	"sessiondb/internal/config"
	"sessiondb/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB(cfg *config.Config) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database")
}

func Migrate() {
	if DB == nil {
		log.Fatal("Database not connected")
	}

	// Enable uuid-ossp extension
	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";")

	err := DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.ApprovalRequest{},

		&models.AuditLog{},
		&models.SavedScript{},
		&models.QueryTab{},
		&models.DBUserCredential{},
		&models.DBInstance{},
		&models.DBTable{},
		&models.DBColumn{},
		&models.DBEntity{},
		&models.DBPrivilege{},
	)
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	SeedRoles()
	log.Println("Database migration completed")
}

func SeedRoles() {
	roles := []string{"Super Admin", "Developer", "Analyst"}
	for _, roleName := range roles {
		var count int64
		DB.Model(&models.Role{}).Where("name = ?", roleName).Count(&count)
		if count == 0 {
			role := models.Role{
				Name:        roleName,
				Description: "Default role",
				IsSystemRole: true,
			}
			DB.Create(&role)
			log.Printf("Seeded role: %s", roleName)
		}
	}
}
