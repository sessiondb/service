package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MetadataRepository struct {
	DB *gorm.DB
}

func NewMetadataRepository(db *gorm.DB) *MetadataRepository {
	return &MetadataRepository{DB: db}
}

func (r *MetadataRepository) SaveTables(tables []models.DBTable) error {
	return r.DB.Save(&tables).Error
}

func (r *MetadataRepository) SaveColumns(columns []models.DBColumn) error {
	return r.DB.Save(&columns).Error
}

func (r *MetadataRepository) SaveEntities(entities []models.DBEntity) error {
	return r.DB.Save(&entities).Error
}

func (r *MetadataRepository) SavePrivileges(privileges []models.DBPrivilege) error {
	return r.DB.Save(&privileges).Error
}

func (r *MetadataRepository) ClearInstanceMetadata(instanceID uuid.UUID) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_id = ?", instanceID).Delete(&models.DBPrivilege{}).Error; err != nil {
			return err
		}
		if err := tx.Where("instance_id = ?", instanceID).Delete(&models.DBEntity{}).Error; err != nil {
			return err
		}
		// Columns have foreign key to tables, so we need to find tables first or delete columns by table join
		// Simpler: delete all columns for tables belonging to this instance
		if err := tx.Exec("DELETE FROM db_columns WHERE table_id IN (SELECT id FROM db_tables WHERE instance_id = ?)", instanceID).Error; err != nil {
			return err
		}
		if err := tx.Where("instance_id = ?", instanceID).Delete(&models.DBTable{}).Error; err != nil {
			return err
		}
		return nil
	})
}
