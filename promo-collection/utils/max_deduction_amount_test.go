package utils

import "testing"

func TestComputeMaxDeductibleAmount(t *testing.T) {
	tests := []struct {
		name       string
		m, p, l, c float64
		expected   float64
	}{
		{
			name:     "m*p is smaller",
			m:        10,
			p:        5,
			l:        20,
			c:        5,
			expected: 50, // min(50, 100)
		},
		{
			name:     "l*c is smaller",
			m:        10,
			p:        20,
			l:        5,
			c:        10,
			expected: 50, // min(200, 50)
		},
		{
			name:     "both equal",
			m:        10,
			p:        5,
			l:        25,
			c:        2,
			expected: 50, // both sides = 50
		},
		{
			name:     "zero values",
			m:        0,
			p:        10,
			l:        5,
			c:        10,
			expected: 0, // min(0, 50)
		},
		{
			name:     "negative values",
			m:        -2,
			p:        5,
			l:        3,
			c:        4,
			expected: -10, // min(-10, 12)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeMaxDeductibleAmount(tt.m, tt.p, tt.l, tt.c)
			if got != tt.expected {
				t.Errorf("ComputeMaxDeductibleAmount(%v,%v,%v,%v) = %v, want %v",
					tt.m, tt.p, tt.l, tt.c, got, tt.expected)
			}
		})
	}
}
