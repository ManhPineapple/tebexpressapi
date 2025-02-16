package dto

type UserAlertPackge struct {
	ID int64 `json:"id"`
	Email string `json:"email"`
	FullName string `json:"full_name"`
	Count int64 `json:"count"`
}
