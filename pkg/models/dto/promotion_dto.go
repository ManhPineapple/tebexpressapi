package dto

type PromotionCustomer struct {
	Name string `json:"name"`
	Prices []PriceLevel `json:"prices"`
	Weights []WeightLevel `json:"weights"`
}

type PriceLevel struct {
	ServiceName string `json:"service_name"`
	Weight float64 `json:"weight"`
	Price float64 `json:"price"`
}
type WeightLevel struct {
	Weight float64 `json:"weight"`
	ErrWeight float64 `json:"err_weight"`
}