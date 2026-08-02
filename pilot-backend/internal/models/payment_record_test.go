package models

import "testing"

func TestPaymentRecord_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*PaymentRecord)
		wantErr error
	}{
		{"valid", func(p *PaymentRecord) {}, nil},
		{"negative amount", func(p *PaymentRecord) { p.Amount = -1 }, ErrInvalidPaymentValue},
		{"negative tolls", func(p *PaymentRecord) { p.Tolls = -1 }, ErrInvalidPaymentValue},
		{"negative taxes", func(p *PaymentRecord) { p.Taxes = -1 }, ErrInvalidPaymentValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PaymentRecord{Amount: 100}
			tt.mutate(&p)
			if err := p.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
