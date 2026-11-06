package domain

// Bike is a single rentable item.
type Bike struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// NewBike creates an available bike. The store assigns the ID.
func NewBike(name string) (Bike, error) {
	bike := Bike{Available: true}
	if err := bike.Rename(name); err != nil {
		return Bike{}, err
	}
	return bike, nil
}

// Rename changes the name. Availability can only change through a rental.
func (s *Bike) Rename(name string) error {
	name, err := required(name, "Bezeichnung")
	if err != nil {
		return err
	}
	s.Name = name
	return nil
}

// rent and release are unexported so that availability only changes together
// with a rental (see NewRental and Rental.Return).
func (s *Bike) rent() error {
	if !s.Available {
		return ErrBikeUnavailable
	}
	s.Available = false
	return nil
}

func (s *Bike) release() {
	s.Available = true
}
