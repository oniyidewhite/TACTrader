package strategy

import (
	"fmt"
	"testing"
)

func TestRoundTo2DecimalPoint(t *testing.T) {
	tests := []struct {
		arg  float64
		want float64
	}{
		{
			arg:  10,
			want: 10,
		},
		{
			arg:  0,
			want: 0,
		},
		{
			arg:  1,
			want: 1,
		},
		{
			arg:  10000000,
			want: 10000000,
		},
		{
			arg:  10.1,
			want: 10.1,
		},
		{
			arg:  10.2,
			want: 10.2,
		},
		{
			arg:  10.12,
			want: 10.12,
		},
		{
			arg:  10.21,
			want: 10.21,
		},
		{
			arg:  10.211,
			want: 10.22,
		},
		{
			arg:  10.213,
			want: 10.22,
		},
		{
			arg:  10.210,
			want: 10.21,
		},
		{
			arg:  10.2111111111111,
			want: 10.22,
		},
		{
			arg:  10.219999999999999,
			want: 10.22,
		},
		{
			arg:  99.991,
			want: 100.00,
		},
		{
			arg:  1002.5677912,
			want: 1002.57,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("TestRoundTo2DecimalPoint(%v) == %v", tt.arg, tt.want), func(t *testing.T) {
			if got := RoundToDecimalPoint(tt.arg, 2); got != tt.want {
				t.Errorf("RoundTo2DecimalPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
