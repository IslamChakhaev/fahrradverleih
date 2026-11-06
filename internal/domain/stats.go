package domain

// Stats holds the key figures shown on the dashboard.
type Stats struct {
	AvailableBikes      int `json:"availableBikes"`
	RentedBikes         int `json:"rentedBikes"`
	TotalBikes          int `json:"totalBikes"`
	RegisteredCustomers int `json:"registeredCustomers"`
}
