package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"time"

	"gorm.io/gorm"
)

type TemplateManager struct {
	db *gorm.DB
}
type TemplateQueryOptions struct {
	UserID    int64
	IDs       []int64
	ID        int64
	IsDefault bool
	Offset    int
	Limit     int
}

func NewTemplateManager(db *gorm.DB) *TemplateManager {
	return &TemplateManager{
		db: db,
	}
}

func (m *TemplateManager) templatesQuery(opts TemplateQueryOptions) *gorm.DB {
	db := m.db
	db = db.Where("status = ? ", constant.TemplateOrderImportActive)

	if len(opts.IDs) > 0 {
		db = db.Where("id IN (?)", opts.IDs)
	}

	if opts.ID > 0 {
		db = db.Where("id = ?", opts.ID)
	}

	if opts.UserID > 0 {
		db = db.Where("user_id = ?", opts.UserID)
	}

	if opts.IsDefault {
		db = db.Where("is_default = ?", constant.TemplateDefault)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	return db
}

func (m *TemplateManager) GetImportPackageTemplate(options TemplateQueryOptions) (*entity.ImportPackageTemplate, error) {
	db := m.templatesQuery(options)
	db = db.Preload("Fields")
	template := &entity.ImportPackageTemplate{}
	db = db.First(template)
	return template, db.Error
}

func (m *TemplateManager) GetImportPackageTemplates(options TemplateQueryOptions) ([]entity.ImportPackageTemplate, error) {
	db := m.templatesQuery(options)
	db = db.Order("is_default DESC,updated_at DESC")
	db = db.Preload("Fields")
	var templates []entity.ImportPackageTemplate
	db = db.Find(&templates)
	return templates, db.Error
}

func (m *TemplateManager) CountImportPackageTemplates(options TemplateQueryOptions) (int64, error) {
	db := m.templatesQuery(options)
	var Count int64
	db = db.Model(&entity.ImportPackageTemplate{}).Count(&Count)
	return Count, db.Error
}

func (m *TemplateManager) SaveImportPackageTemplate(template entity.ImportPackageTemplate, fields []entity.ImportTemplateField) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if template.IsDefault == constant.TemplateDefault {
		err := tx.Model(&entity.ImportPackageTemplate{}).Where("user_id = ?", template.UserID).Updates(
			map[string]interface{}{
				"is_default": constant.TemplateIsNotDefault,
				"updated_at": time.Now(),
			}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err := tx.Save(&template).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, field := range fields {
		field.TemplateID = template.ID
		err := tx.Save(&field).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m *TemplateManager) UpdateImportPackageTemplate(template *entity.ImportPackageTemplate, fields []entity.ImportTemplateField) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if template.IsDefault == constant.TemplateDefault {
		err := tx.Model(&entity.ImportPackageTemplate{}).Where("user_id = ?", template.UserID).Updates(
			map[string]interface{}{
				"is_default": constant.TemplateIsNotDefault,
				"updated_at": time.Now(),
			}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err := tx.Save(&template).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&entity.ImportTemplateField{}).Where("template_id IN (?)", template.ID).Delete(&entity.ImportTemplateField{}).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, field := range fields {
		err := tx.Save(&field).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m *TemplateManager) DeleteTemplates(ids []int64) error {
	db := m.db
	db = db.Model(&entity.ImportPackageTemplate{}).Where("id IN (?)", ids).Updates(map[string]interface{}{
		"status":     constant.TemplateOrderImportInactive,
		"updated_at": time.Now(),
	})
	return db.Error
}
