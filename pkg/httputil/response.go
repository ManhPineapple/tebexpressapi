package httputil

import "encoding/json"

type ErrorResponse struct {
	Error    string   `json:"error"`
	Messages []string `json:"messages,omitempty"`
	Message  string   `json:"message,omitempty"`
}

func Transform(data, transform interface{}) error {
	by, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(by, transform)
}
