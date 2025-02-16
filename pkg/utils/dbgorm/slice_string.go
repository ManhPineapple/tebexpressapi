package dbgorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type SliceString []string

func (fv SliceString) Value() (driver.Value, error) {
	b, _ := json.Marshal(fv)
	return string(b), nil
}

func (fv *SliceString) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid Scan Source")
	}

	sl := make([]string, 0)
	_ = json.Unmarshal(s, &sl)
	*fv = sl

	return nil
}
