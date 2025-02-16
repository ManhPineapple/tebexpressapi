package utils

import "encoding/json"

func TypeConverter(data interface{},result interface{})  error{
	b, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, result)
	if err != nil {
		return err
	}
	return nil
}
