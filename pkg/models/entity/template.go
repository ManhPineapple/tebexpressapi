package entity

import "tebexpressapi/pkg/utils/dbgorm"

type ImportPackageTemplate struct {
	dbgorm.Model
	Name         string                `json:"name"`
	File         string                `json:"file"`
	Status       int                   `json:"status"`
	IsDefault    int                   `json:"is_default"`
	UserID       int64                 `json:"user_id"`
	ImportFields string                `json:"import_fields"`
	Fields       []ImportTemplateField `json:"fields" gorm:"save_associations:false;foreignKey:TemplateID"`
}

type ImportTemplateField struct {
	dbgorm.Model
	FieldName       string `json:"field_name"`
	FieldAssignName string `json:"field_assign_name"`
	FieldKey        int    `json:"field_key"`
	FieldAssignKey  int    `json:"field_assign_key"`
	TemplateID      int64  `json:"template_id"`
}
