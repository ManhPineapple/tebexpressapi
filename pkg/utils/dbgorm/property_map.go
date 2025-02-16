package dbgorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

type PropertyMap map[string]interface{}

func (fv PropertyMap) Value() (driver.Value, error) {
	b, _ := json.Marshal(fv)
	return string(b), nil
}

func (fv *PropertyMap) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid Scan Source")
	}

	sl := make(map[string]interface{}, 0)
	_ = json.Unmarshal(s, &sl)
	*fv = sl

	return nil
}

func (fv *PropertyMap) Raw() string {
	value, err := fv.Value()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	attribute := cast.ToStringMapString(value)
	tmp := make([]string, 0)

	for key, val := range attribute {
		tmp = append(tmp, key+":"+val)
	}

	return strings.Join(tmp, " | ")
}

func (fv *PropertyMap) GetValue(key string) string {
	value, err := fv.Value()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	attribute := cast.ToStringMapString(value)

	return attribute[key]

}
