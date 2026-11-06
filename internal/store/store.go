// Package store keeps all data in memory and is safe for concurrent use.
// Methods return copies, so callers cannot change stored data directly.
package store

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"fahrradverleih/internal/domain"
)

// Store holds all bikes, customers, tariffs and rentals.
type Store struct {
	mu sync.RWMutex

	bikes     map[int]domain.Bike
	customers map[int]domain.Customer
	tariffs   map[int]domain.Tariff
	rentals   map[int]domain.Rental

	nextBikeID     int
	nextCustomerID int
	nextTariffID   int
	nextRentalID   int
}

// New returns an empty store.
func New() *Store {
	return &Store{
		bikes:          make(map[int]domain.Bike),
		customers:      make(map[int]domain.Customer),
		tariffs:        make(map[int]domain.Tariff),
		rentals:        make(map[int]domain.Rental),
		nextBikeID:     1,
		nextCustomerID: 1,
		nextTariffID:   1,
		nextRentalID:   1,
	}
}

func notFound(label string, id int) error {
	return fmt.Errorf("%w: %s mit ID %d", domain.ErrNotFound, label, id)
}

// sortedValues returns the map values ordered by ID, because map iteration
// order is random. The result is never nil, so it encodes as a JSON array.
func sortedValues[T any](items map[int]T) []T {
	ids := make([]int, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	values := make([]T, 0, len(ids))
	for _, id := range ids {
		values = append(values, items[id])
	}
	return values
}

// Bikes returns all bikes ordered by ID.
func (s *Store) Bikes() []domain.Bike {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.bikes)
}

// AddBike creates a new bike.
func (s *Store) AddBike(name string) (domain.Bike, error) {
	bike, err := domain.NewBike(name)
	if err != nil {
		return domain.Bike{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	bike.ID = s.nextBikeID
	s.nextBikeID++
	s.bikes[bike.ID] = bike
	return bike, nil
}

// UpdateBike changes the name of a bike.
func (s *Store) UpdateBike(id int, name string) (domain.Bike, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bike, ok := s.bikes[id]
	if !ok {
		return domain.Bike{}, notFound("Fahrrad", id)
	}
	if err := bike.Rename(name); err != nil {
		return domain.Bike{}, err
	}
	s.bikes[id] = bike
	return bike, nil
}

// Customers returns all customers ordered by ID.
func (s *Store) Customers() []domain.Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.customers)
}

// AddCustomer registers a new customer.
func (s *Store) AddCustomer(name, idType, idNumber string) (domain.Customer, error) {
	customer, err := domain.NewCustomer(name, idType, idNumber)
	if err != nil {
		return domain.Customer{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	customer.ID = s.nextCustomerID
	s.nextCustomerID++
	s.customers[customer.ID] = customer
	return customer, nil
}

// UpdateCustomer replaces the data of a customer.
func (s *Store) UpdateCustomer(id int, name, idType, idNumber string) (domain.Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	customer, ok := s.customers[id]
	if !ok {
		return domain.Customer{}, notFound("Person", id)
	}
	if err := customer.Update(name, idType, idNumber); err != nil {
		return domain.Customer{}, err
	}
	s.customers[id] = customer
	return customer, nil
}

// Tariffs returns all tariffs ordered by ID.
func (s *Store) Tariffs() []domain.Tariff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.tariffs)
}

// Tariff returns a single tariff.
func (s *Store) Tariff(id int) (domain.Tariff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tariff, ok := s.tariffs[id]
	if !ok {
		return domain.Tariff{}, notFound("Tarif", id)
	}
	return tariff, nil
}

// AddTariff creates a new tariff.
func (s *Store) AddTariff(name string, hourlyPriceCents int64) (domain.Tariff, error) {
	tariff, err := domain.NewTariff(name, hourlyPriceCents)
	if err != nil {
		return domain.Tariff{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tariff.ID = s.nextTariffID
	s.nextTariffID++
	s.tariffs[tariff.ID] = tariff
	return tariff, nil
}

// UpdateTariff changes a tariff. Existing rentals keep their copied prices.
func (s *Store) UpdateTariff(id int, name string, hourlyPriceCents int64) (domain.Tariff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tariff, ok := s.tariffs[id]
	if !ok {
		return domain.Tariff{}, notFound("Tarif", id)
	}
	if err := tariff.Update(name, hourlyPriceCents); err != nil {
		return domain.Tariff{}, err
	}
	s.tariffs[id] = tariff
	return tariff, nil
}

// Rentals returns all rentals with their display data, ordered by ID.
func (s *Store) Rentals() []domain.RentalDetails {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rentals := sortedValues(s.rentals)
	details := make([]domain.RentalDetails, 0, len(rentals))
	for _, rental := range rentals {
		details = append(details, s.details(rental))
	}
	return details
}

// details must be called with the lock held. Bikes and customers cannot be
// deleted, so the lookups always succeed.
func (s *Store) details(rental domain.Rental) domain.RentalDetails {
	return rental.Details(s.customers[rental.CustomerID], s.bikes[rental.BikeID])
}

// CreateRental rents a bike. All checks and changes happen under one lock, so a
// bike can never be rented twice at the same time.
func (s *Store) CreateRental(customerID, bikeID, tariffID int, start time.Time, durationHours int) (domain.RentalDetails, error) {
	if err := domain.ValidateRentalSelection(customerID, bikeID, tariffID); err != nil {
		return domain.RentalDetails{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	customer, ok := s.customers[customerID]
	if !ok {
		return domain.RentalDetails{}, notFound("Person", customerID)
	}
	bike, ok := s.bikes[bikeID]
	if !ok {
		return domain.RentalDetails{}, notFound("Fahrrad", bikeID)
	}
	tariff, ok := s.tariffs[tariffID]
	if !ok {
		return domain.RentalDetails{}, notFound("Tarif", tariffID)
	}

	rental, err := domain.NewRental(customer, &bike, tariff, start, durationHours)
	if err != nil {
		return domain.RentalDetails{}, err
	}
	rental.ID = s.nextRentalID
	s.nextRentalID++
	s.rentals[rental.ID] = rental
	s.bikes[bikeID] = bike
	return s.details(rental), nil
}

// ReturnRental ends a rental and makes its bike available in one step.
func (s *Store) ReturnRental(id int, at time.Time) (domain.RentalDetails, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rental, ok := s.rentals[id]
	if !ok {
		return domain.RentalDetails{}, notFound("Ausleihe", id)
	}
	bike := s.bikes[rental.BikeID]
	if err := rental.Return(&bike, at); err != nil {
		return domain.RentalDetails{}, err
	}
	s.rentals[id] = rental
	s.bikes[bike.ID] = bike
	return s.details(rental), nil
}

// Stats counts the current data under one lock, so the figures always match.
func (s *Store) Stats() domain.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := domain.Stats{
		TotalBikes:          len(s.bikes),
		RegisteredCustomers: len(s.customers),
	}
	for _, bike := range s.bikes {
		if bike.Available {
			stats.AvailableBikes++
		} else {
			stats.RentedBikes++
		}
	}
	return stats
}
