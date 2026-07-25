package domain

import (
	"math"
	"strconv"
)

const (
	defaultBufferBasisPoints = 2000
	basisPointsPerPercent    = 100
	totalBasisPoints         = 10000
)

type DailyCapacity struct {
	halfHours uint8
}

func NewDailyCapacity(hours float64) (DailyCapacity, error) {
	if math.IsNaN(hours) || math.IsInf(hours, 0) || hours <= 0 {
		return DailyCapacity{}, ErrDailyCapacityNotPositive
	}
	if hours > 24 {
		return DailyCapacity{}, ErrDailyCapacityExceedsLimit
	}
	halfHours := hours * 2
	if math.Abs(halfHours-math.Round(halfHours)) > 1e-9 {
		return DailyCapacity{}, ErrDailyCapacityInvalidIncrement
	}
	return DailyCapacity{halfHours: uint8(math.Round(halfHours))}, nil
}

func (capacity DailyCapacity) Hours() float64 {
	return float64(capacity.halfHours) / 2
}

func (capacity DailyCapacity) Decimal() string {
	return strconv.FormatFloat(capacity.Hours(), 'f', 1, 64)
}

type BufferPercentage struct {
	basisPoints uint16
}

func DefaultBufferPercentage() BufferPercentage {
	return BufferPercentage{basisPoints: defaultBufferBasisPoints}
}

func NewBufferPercentage(percentage float64) (BufferPercentage, error) {
	if math.IsNaN(percentage) || math.IsInf(percentage, 0) ||
		percentage < 0 || percentage >= 100 {
		return BufferPercentage{}, ErrBufferOutOfRange
	}
	basisPoints := percentage * basisPointsPerPercent
	if math.Abs(basisPoints-math.Round(basisPoints)) > 1e-9 {
		return BufferPercentage{}, ErrBufferPrecision
	}
	return BufferPercentage{basisPoints: uint16(math.Round(basisPoints))}, nil
}

func (buffer BufferPercentage) Percentage() float64 {
	return float64(buffer.basisPoints) / basisPointsPerPercent
}

func (buffer BufferPercentage) Decimal() string {
	return strconv.FormatFloat(buffer.Percentage(), 'f', 2, 64)
}

func CommitmentCapacity(
	dailyCapacity DailyCapacity,
	buffer BufferPercentage,
) float64 {
	numerator := int(dailyCapacity.halfHours) *
		(totalBasisPoints - int(buffer.basisPoints))
	roundedHalfHours := (numerator + totalBasisPoints/2) / totalBasisPoints
	return float64(roundedHalfHours) / 2
}
