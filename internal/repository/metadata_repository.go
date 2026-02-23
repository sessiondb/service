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
	if len(tables) == 0 {
		return nil
	}
	return r.DB.Save(&tables).Error
}

func (r *MetadataRepository) SaveColumns(columns []models.DBColumn) error {
	if len(columns) == 0 {
		return nil
	}
	return r.DB.Save(&columns).Error
}

func (r *MetadataRepository) SaveEntities(entities []models.DBEntity) error {
	if len(entities) == 0 {
		return nil
	}
	// Deduplicate within the batch by (InstanceID, Name, Type) before inserting.
	// pg_roles / mysql.user can return duplicate rows in a single query result.
	seen := make(map[string]struct{}, len(entities))
	unique := entities[:0]
	for _, e := range entities {
		key := e.InstanceID.String() + "|" + e.Name + "|" + e.Type
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			unique = append(unique, e)
		}
	}
	return r.DB.Create(&unique).Error
}

func (r *MetadataRepository) SavePrivileges(privileges []models.DBPrivilege) error {
	if len(privileges) == 0 {
		return nil
	}
	return r.DB.Save(&privileges).Error
}

func (r *MetadataRepository) ClearInstanceMetadata(instanceID uuid.UUID) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// Hard-delete privileges and entities so fresh syncs can re-insert cleanly.
		// Using Unscoped() bypasses GORM's soft-delete so rows are truly removed.
		if err := tx.Unscoped().Where("instance_id = ?", instanceID).Delete(&models.DBPrivilege{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("instance_id = ?", instanceID).Delete(&models.DBEntity{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("instance_id = ?", instanceID).Delete(&models.DBRoleMembership{}).Error; err != nil {
			return err
		}
		// Hard-delete columns and tables the same way.
		if err := tx.Exec("DELETE FROM db_columns WHERE table_id IN (SELECT id FROM db_tables WHERE instance_id = ?)", instanceID).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("instance_id = ?", instanceID).Delete(&models.DBTable{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *MetadataRepository) SaveRoleMemberships(memberships []models.DBRoleMembership) error {
	if len(memberships) == 0 {
		return nil
	}
	return r.DB.Create(&memberships).Error
}

func (r *MetadataRepository) FindRoleMembershipsByMember(instanceID uuid.UUID, member string) ([]string, error) {
	var roles []string
	err := r.DB.Model(&models.DBRoleMembership{}).
		Where("instance_id = ? AND member_name = ?", instanceID, member).
		Pluck("role_name", &roles).Error
	return roles, err
}

func (r *MetadataRepository) FindPrivilegesByGrantee(instanceID uuid.UUID, grantee string) ([]models.DBPrivilege, error) {
	var privs []models.DBPrivilege
	err := r.DB.Where("instance_id = ? AND grantee = ?", instanceID, grantee).Find(&privs).Error
	return privs, err
}
func (r *MetadataRepository) GetDatabases(instanceID uuid.UUID) ([]string, error) {
	var databases []string
	err := r.DB.Model(&models.DBTable{}).
		Where("instance_id = ?", instanceID).
		Distinct("database").
		Pluck("database", &databases).Error
	return databases, err
}

func (r *MetadataRepository) GetTables(instanceID uuid.UUID, database string) ([]models.DBTable, error) {
	var tables []models.DBTable
	err := r.DB.Where("instance_id = ? AND database = ?", instanceID, database).Find(&tables).Error
	return tables, err
}

func (r *MetadataRepository) GetTableByID(tableID uuid.UUID) (*models.DBTable, error) {
	var table models.DBTable
	err := r.DB.Preload("Columns").First(&table, "id = ?", tableID).Error
	return &table, err
}
func (r *MetadataRepository) GetFullSchema(instanceID uuid.UUID) ([]models.DBTable, error) {
	var tables []models.DBTable
	err := r.DB.Preload("Columns").Where("instance_id = ?", instanceID).Find(&tables).Error
	return tables, err
}

// FindAllEntities returns one DBEntity record per (instance_id, name) combination, ordered by name.
// Uses a raw DISTINCT ON query so stale duplicates are never returned even if any survived in the DB.
func (r *MetadataRepository) FindAllEntities(entityType string) ([]models.DBEntity, error) {
	var entities []models.DBEntity
	query := `
		SELECT DISTINCT ON (instance_id, name) *
		FROM db_entities
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	if entityType != "" {
		query += " AND type = $1"
		args = append(args, entityType)
	}
	query += " ORDER BY instance_id, name, created_at ASC"
	err := r.DB.Raw(query, args...).Scan(&entities).Error
	return entities, err
}

// FindEntitiesByInstance returns DBEntity records for a specific instance, filtered by type.
func (r *MetadataRepository) FindEntitiesByInstance(instanceID uuid.UUID, entityType string) ([]models.DBEntity, error) {
	var entities []models.DBEntity
	q := r.DB.Where("instance_id = ?", instanceID).Order("name asc")
	if entityType != "" {
		q = q.Where("type = ?", entityType)
	}
	err := q.Find(&entities).Error
	return entities, err
}

// CountMembersByRole counts how many members have a given role in a specific instance.
func (r *MetadataRepository) CountMembersByRole(instanceID uuid.UUID, roleName string) (int64, error) {
	var count int64
	err := r.DB.Model(&models.DBRoleMembership{}).
		Where("instance_id = ? AND role_name = ?", instanceID, roleName).
		Count(&count).Error
	return count, err
}
