package domain

import (
	"fmt"
	"math"
)

// GramCatchVariance stores meters from the landing target in millimeters.
type GramCatchVariance int64

func CatchVarianceFromKilograms(value float64) (GramCatchVariance, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, FieldError{Field: "catch_variance", Message: "must be finite"}
	}
	if value < -196 || value > 100 {
		return 0, FieldError{Field: "catch_variance", Message: "outside supported range"}
	}
	return GramCatchVariance(math.Round(value * 1000)), nil
}

func (t GramCatchVariance) Float64() float64 { return float64(t) / 1000 }

func (t GramCatchVariance) String() string { return fmt.Sprintf("%.3f", t.Float64()) }

type CatchVarianceEnvelope struct {
	Minimum GramCatchVariance `json:"minimum"`
	Maximum GramCatchVariance `json:"maximum"`
}

func NewCatchVarianceEnvelope(minimum, maximum GramCatchVariance) (CatchVarianceEnvelope, error) {
	if minimum >= maximum {
		return CatchVarianceEnvelope{}, FieldError{Field: "catch_variance_range", Message: "minimum must be lower than maximum"}
	}
	return CatchVarianceEnvelope{Minimum: minimum, Maximum: maximum}, nil
}

func (r CatchVarianceEnvelope) Contains(value GramCatchVariance) bool {
	return value >= r.Minimum && value <= r.Maximum
}

func (r CatchVarianceEnvelope) Validate() error {
	if r.Minimum >= r.Maximum {
		return FieldError{Field: "catch_variance_range", Message: "minimum must be lower than maximum"}
	}
	return nil
}
