package sqlmanager

import (
	"tebexpressapi/pkg/models/entity"

	"gorm.io/gorm"
)

type StateManager struct {
	db *gorm.DB
}

func NewStateManager(db *gorm.DB) *StateManager {
	return &StateManager{
		db: db,
	}
}

type StateOption struct {
	Code      string
	Country   string
	Countries []string
	Status    int
}

func (m *StateManager) BuildStateQuery(opts StateOption) *gorm.DB {
	db := m.db
	if opts.Country != "" {
		db = db.Where("country = ?", opts.Country)
	}
	if opts.Code != "" {
		db = db.Where("code = ? OR UPPER(name) = ? ", opts.Code, opts.Code)
	}
	if opts.Status > 0 {
		db = db.Where("status = ?", opts.Status)
	}
	if len(opts.Countries) > 0 {
		db = db.Where("country IN (?)", opts.Countries)
	}
	db = db.Order("id DESC")
	return db
}

func (m *StateManager) GetStates(opts StateOption) ([]*entity.State, error) {
	db := m.BuildStateQuery(opts)
	var states []*entity.State
	db = db.Find(&states)
	return states, db.Error
}

func (m *StateManager) GetState(opts StateOption) (*entity.State, error) {
	db := m.BuildStateQuery(opts)
	state := &entity.State{}
	db = db.First(state)
	return state, db.Error
}
