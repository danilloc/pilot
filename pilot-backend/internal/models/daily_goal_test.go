package models

import "testing"

func TestDailyGoal_Validate(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		wantErr error
	}{
		{"valid", 350.00, nil},
		{"zero", 0, ErrInvalidGoalAmount},
		{"negative", -10, ErrInvalidGoalAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := DailyGoal{GoalAmount: tt.amount}
			if err := g.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
