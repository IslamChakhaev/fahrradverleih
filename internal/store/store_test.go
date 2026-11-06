package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"fahrradverleih/internal/domain"
)

var testStart = time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

// fixture contains one bike, one customer and one tariff, all with ID 1.
func fixture(t *testing.T) *Store {
	t.Helper()
	s := New()
	if _, err := s.AddBike("Citybike 52"); err != nil {
		t.Fatalf("AddBike: %v", err)
	}
	if _, err := s.AddCustomer("Max Mustermann", "Reisepass", "C0000001"); err != nil {
		t.Fatalf("AddCustomer: %v", err)
	}
	if _, err := s.AddTariff("Standard", 500); err != nil {
		t.Fatalf("AddTariff: %v", err)
	}
	return s
}

func assertStats(t *testing.T, s *Store, want domain.Stats) {
	t.Helper()
	if got := s.Stats(); got != want {
		t.Errorf("Stats() = %+v, want %+v", got, want)
	}
}

func TestAddAssignsUniqueSortedIDs(t *testing.T) {
	s := New()
	for _, name := range []string{"A", "B", "C"} {
		if _, err := s.AddBike(name); err != nil {
			t.Fatalf("AddBike: %v", err)
		}
	}
	bikes := s.Bikes()
	if len(bikes) != 3 {
		t.Fatalf("got %d bikes, want 3", len(bikes))
	}
	for i, bike := range bikes {
		if bike.ID != i+1 {
			t.Errorf("bikes[%d].ID = %d, want %d", i, bike.ID, i+1)
		}
	}
}

func TestListsAreEmptyNotNil(t *testing.T) {
	s := New()
	if s.Bikes() == nil || s.Customers() == nil || s.Tariffs() == nil || s.Rentals() == nil {
		t.Error("empty lists must not be nil")
	}
}

func TestReturnedSlicesAreCopies(t *testing.T) {
	s := fixture(t)
	s.Bikes()[0].Name = "changed"
	if got := s.Bikes()[0].Name; got != "Citybike 52" {
		t.Errorf("stored name changed to %q", got)
	}
}

func TestAddRejectsInvalidInputWithoutConsumingData(t *testing.T) {
	s := New()
	if _, err := s.AddBike(""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("AddBike: got %v, want ErrInvalidInput", err)
	}
	if _, err := s.AddCustomer("", "", ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("AddCustomer: got %v, want ErrInvalidInput", err)
	}
	if _, err := s.AddTariff("Standard", -1); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("AddTariff: got %v, want ErrInvalidInput", err)
	}
	assertStats(t, s, domain.Stats{})
}

func TestUpdateBikeKeepsAvailability(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 2); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}

	bike, err := s.UpdateBike(1, "Citybike 56")
	if err != nil {
		t.Fatalf("UpdateBike: %v", err)
	}
	if bike.Name != "Citybike 56" || bike.Available {
		t.Errorf("got %+v, want new name and still rented", bike)
	}
	if got := s.Rentals()[0].BikeName; got != "Citybike 56" {
		t.Errorf("rental shows bike name %q", got)
	}
}

func TestUpdateCustomer(t *testing.T) {
	s := fixture(t)
	customer, err := s.UpdateCustomer(1, "Erika Musterfrau", "Personalausweis", "T0000002")
	if err != nil {
		t.Fatalf("UpdateCustomer: %v", err)
	}
	if customer.ID != 1 || customer.Name != "Erika Musterfrau" || customer.IDNumber != "T0000002" {
		t.Errorf("got %+v", customer)
	}
}

func TestUpdateErrors(t *testing.T) {
	s := fixture(t)
	tests := []struct {
		name string
		call func() error
		want error
	}{
		{"unknown bike", func() error { _, err := s.UpdateBike(99, "X"); return err }, domain.ErrNotFound},
		{"invalid bike", func() error { _, err := s.UpdateBike(1, ""); return err }, domain.ErrInvalidInput},
		{"unknown customer", func() error { _, err := s.UpdateCustomer(99, "X", "Y", "Z"); return err }, domain.ErrNotFound},
		{"invalid customer", func() error { _, err := s.UpdateCustomer(1, "", "Y", "Z"); return err }, domain.ErrInvalidInput},
		{"unknown tariff", func() error { _, err := s.UpdateTariff(99, "X", 1); return err }, domain.ErrNotFound},
		{"invalid tariff", func() error { _, err := s.UpdateTariff(1, "X", -1); return err }, domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
	if tariff, _ := s.Tariff(1); tariff.HourlyPriceCents != 500 {
		t.Errorf("failed update changed price to %d", tariff.HourlyPriceCents)
	}
}

func TestCreateRentalUpdatesAvailabilityAndStats(t *testing.T) {
	s := fixture(t)

	rental, err := s.CreateRental(1, 1, 1, testStart, 3)
	if err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	if rental.ID != 1 || rental.CustomerName != "Max Mustermann" || rental.BikeName != "Citybike 52" {
		t.Errorf("wrong details: %+v", rental)
	}
	if rental.TotalPriceCents != 1500 || !rental.Active {
		t.Errorf("wrong price or status: %+v", rental)
	}
	if s.Bikes()[0].Available {
		t.Error("bike should be rented")
	}
	assertStats(t, s, domain.Stats{RentedBikes: 1, TotalBikes: 1, RegisteredCustomers: 1})
}

func TestCreateRentalErrors(t *testing.T) {
	tests := []struct {
		name                         string
		customerID, bikeID, tariffID int
		hours                        int
		want                         error
	}{
		{"unknown customer", 99, 1, 1, 2, domain.ErrNotFound},
		{"unknown bike", 1, 99, 1, 2, domain.ErrNotFound},
		{"unknown tariff", 1, 1, 99, 2, domain.ErrNotFound},
		{"missing customer", 0, 1, 1, 2, domain.ErrInvalidInput},
		{"invalid duration", 1, 1, 1, 0, domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(t)
			_, err := s.CreateRental(tt.customerID, tt.bikeID, tt.tariffID, testStart, tt.hours)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			assertStats(t, s, domain.Stats{AvailableBikes: 1, TotalBikes: 1, RegisteredCustomers: 1})
			if len(s.Rentals()) != 0 {
				t.Error("failed rental must not be stored")
			}
		})
	}
}

func TestBikeCannotBeRentedTwice(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 2); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	if _, err := s.CreateRental(1, 1, 1, testStart, 2); !errors.Is(err, domain.ErrBikeUnavailable) {
		t.Errorf("got %v, want ErrBikeUnavailable", err)
	}
	if got := len(s.Rentals()); got != 1 {
		t.Errorf("got %d rentals, want 1", got)
	}
}

func TestReturnRental(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 3); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}

	rental, err := s.ReturnRental(1, testStart.Add(time.Hour))
	if err != nil {
		t.Fatalf("ReturnRental: %v", err)
	}
	if rental.Active || rental.ReturnedAt == nil {
		t.Errorf("rental should be returned: %+v", rental)
	}
	assertStats(t, s, domain.Stats{AvailableBikes: 1, TotalBikes: 1, RegisteredCustomers: 1})

	if _, err := s.ReturnRental(1, testStart); !errors.Is(err, domain.ErrAlreadyReturned) {
		t.Errorf("second return: got %v, want ErrAlreadyReturned", err)
	}
	if _, err := s.ReturnRental(99, testStart); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown rental: got %v, want ErrNotFound", err)
	}
}

func TestSecondReturnDoesNotReleaseNewRental(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 1); err != nil {
		t.Fatalf("first rental: %v", err)
	}
	if _, err := s.ReturnRental(1, testStart); err != nil {
		t.Fatalf("return: %v", err)
	}
	if _, err := s.CreateRental(1, 1, 1, testStart, 1); err != nil {
		t.Fatalf("second rental: %v", err)
	}

	if _, err := s.ReturnRental(1, testStart); !errors.Is(err, domain.ErrAlreadyReturned) {
		t.Fatalf("got %v, want ErrAlreadyReturned", err)
	}
	if s.Bikes()[0].Available {
		t.Error("bike of the second rental must stay rented")
	}
}

func TestTariffChangeKeepsRentalPrice(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 3); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}

	if _, err := s.UpdateTariff(1, "Hochsaison", 900); err != nil {
		t.Fatalf("UpdateTariff: %v", err)
	}

	rental := s.Rentals()[0]
	if rental.TariffName != "Standard" || rental.HourlyPriceCents != 500 || rental.TotalPriceCents != 1500 {
		t.Errorf("stored rental changed: %+v", rental)
	}
	tariff, err := s.Tariff(1)
	if err != nil {
		t.Fatalf("Tariff: %v", err)
	}
	if tariff.Name != "Hochsaison" || tariff.HourlyPriceCents != 900 {
		t.Errorf("tariff not updated: %+v", tariff)
	}
}

func TestConcurrentRentalsOfSameBike(t *testing.T) {
	s := fixture(t)
	const attempts = 100

	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CreateRental(1, 1, 1, testStart, 1)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	successes := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case !errors.Is(err, domain.ErrBikeUnavailable):
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("%d rentals succeeded, want exactly 1", successes)
	}
	assertStats(t, s, domain.Stats{RentedBikes: 1, TotalBikes: 1, RegisteredCustomers: 1})
}

func TestSeed(t *testing.T) {
	s := New()
	if err := s.Seed(testStart); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	assertStats(t, s, domain.Stats{AvailableBikes: 17, RentedBikes: 3, TotalBikes: 20, RegisteredCustomers: 10})
	if got := len(s.Tariffs()); got != 3 {
		t.Errorf("got %d tariffs, want 3", got)
	}
	if got := len(s.Rentals()); got != 3 {
		t.Errorf("got %d rentals, want 3", got)
	}
}
