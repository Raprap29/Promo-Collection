package utils

import "math"

func ComputeMaxDeductibleAmount(
	maxDataCollectionPercent, promoDataAllocated, loanBalance, conversationRate float64) float64 {
	return math.Min(maxDataCollectionPercent*promoDataAllocated, loanBalance*conversationRate)
}
