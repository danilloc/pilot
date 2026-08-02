package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTrip_Validate(t *testing.T) {
	base := func() Trip {
		start := time.Now()
		return Trip{DistanceKM: 10, FareValue: 25.5, StartedAt: start, EndedAt: start.Add(20 * time.Minute)}
	}

	tests := []struct {
		name    string
		mutate  func(*Trip)
		wantErr error
	}{
		{"valid", func(tr *Trip) {}, nil},
		{"zero distance", func(tr *Trip) { tr.DistanceKM = 0 }, ErrInvalidDistance},
		{"negative distance", func(tr *Trip) { tr.DistanceKM = -1 }, ErrInvalidDistance},
		{"negative fare", func(tr *Trip) { tr.FareValue = -0.01 }, ErrInvalidFare},
		{"zero fare allowed", func(tr *Trip) { tr.FareValue = 0 }, nil},
		{"end before start", func(tr *Trip) { tr.EndedAt = tr.StartedAt.Add(-time.Minute) }, ErrInvalidTimeRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := base()
			tt.mutate(&trip)
			err := trip.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTrip_JSONMarshal_OmitsCoordinates(t *testing.T) {
	trip := Trip{
		DistanceKM: 5,
		FareValue:  20,
		StartLat:   -23.55,
		StartLng:   -46.63,
		EndLat:     -23.56,
		EndLng:     -46.64,
	}

	out, err := json.Marshal(trip)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	for _, field := range []string{"start_latitude", "start_longitude", "end_latitude", "end_longitude"} {
		if strings.Contains(string(out), field) {
			t.Errorf("JSON output leaked coordinate field %q: %s", field, out)
		}
	}
}
