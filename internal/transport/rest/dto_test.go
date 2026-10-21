package rest

import (
	"encoding/json"
	"testing"
)

func TestMoneyUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Money
		wantErr bool
	}{
		{name: "whole rubles", input: `350`, want: 35000},
		{name: "rubles and kopecks", input: `123.45`, want: 12345},
		{name: "one decimal place", input: `12.5`, want: 1250},
		{name: "zero", input: `0`, wantErr: true},
		{name: "negative", input: `-10`, wantErr: true},
		{name: "too many decimals", input: `12.345`, wantErr: true},
		{name: "string instead of number", input: `"12.50"`, wantErr: true},
		{name: "null", input: `null`, wantErr: true},
		{name: "invalid value", input: `abc`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Money
			err := json.Unmarshal([]byte(tt.input), &got)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %d kopecks, want %d", got, tt.want)
			}
		})
	}
}

func TestMoneyMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input Money
		want  string
	}{
		{name: "whole rubles", input: 35000, want: `350.00`},
		{name: "with kopecks", input: 12345, want: `123.45`},
		{name: "less than one ruble", input: 50, want: `0.50`},
		{name: "zero", input: 0, want: `0.00`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMoneyMarshalNegativeValue(t *testing.T) {
	_, err := json.Marshal(Money(-100))
	if err == nil {
		t.Fatal("expected an error for negative amount")
	}
}
