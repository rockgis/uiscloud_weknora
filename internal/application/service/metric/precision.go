package metric

import (
	"github.com/Tencent/WeKnora/internal/types"
)

// PrecisionMetric calculates precision for retrieval evaluation
type PrecisionMetric struct{}

// NewPrecisionMetric creates a new PrecisionMetric instance
func NewPrecisionMetric() *PrecisionMetric {
	return &PrecisionMetric{}
}

// Compute calculates the precision score
func (r *PrecisionMetric) Compute(metricInput *types.MetricInput) float64 {
	// Get ground truth and predicted IDs
	gts := metricInput.RetrievalGT
	ids := metricInput.RetrievalIDs

	// Handle edge cases
	if len(gts) == 0 || len(ids) == 0 {
		return 0.0
	}

	// Convert ground truth to sets for efficient lookup
	gtSets := SliceMap(gts, ToSet)
	// Count total hits across all queries
	ahit := Fold(gtSets, 0, func(a int, b map[int]struct{}) int { return a + Hit(ids, b) })

	// Average per-query precision: sum(hits_per_query / len(ids)) / len(gts)
	return float64(ahit) / float64(len(ids)*len(gts))
}
