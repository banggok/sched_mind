package domain

import (
	"errors"
	"fmt"
	"math/big"
	"time"
)

var (
	ErrInvalidDecimal     = errors.New("invalid scheduling decimal")
	ErrDataIntegrity      = errors.New("scheduling data integrity violation")
	ErrNoConvergence      = errors.New("automatic dependency reconciliation did not converge")
	ErrConcurrentConflict = errors.New("schedule changed concurrently")
)

type Timeline string

const (
	Execution  Timeline = "execution"
	Commitment Timeline = "commitment"
	Actual     Timeline = "actual"
)

func BAUCapacityMinutes(resolvedHours *big.Rat) *big.Rat {
	if resolvedHours == nil || resolvedHours.Sign() <= 0 {
		return new(big.Rat)
	}
	return new(big.Rat).Mul(resolvedHours, big.NewRat(60, 1))
}

func ParseDecimal(value string) (*big.Rat, error) {
	decimal, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrInvalidDecimal, value)
	}
	return decimal, nil
}

func CapacityMinutes(
	resolvedHours *big.Rat,
	memberBufferPercentage string,
	projectBufferPercentage int,
	timeline Timeline,
) (*big.Rat, error) {
	if resolvedHours == nil || resolvedHours.Sign() <= 0 {
		return new(big.Rat), nil
	}
	memberBuffer, err := ParseDecimal(memberBufferPercentage)
	if err != nil {
		return nil, err
	}
	if memberBuffer.Sign() < 0 || memberBuffer.Cmp(big.NewRat(100, 1)) >= 0 {
		return nil, ErrDataIntegrity
	}
	rawExecutionHours := new(big.Rat).Mul(
		resolvedHours,
		percentFactor(memberBuffer),
	)
	if timeline == Execution {
		return roundHoursToHalfHourMinutes(rawExecutionHours), nil
	}
	if projectBufferPercentage < 0 || projectBufferPercentage > 100 {
		return nil, ErrDataIntegrity
	}
	rawCommitmentHours := new(big.Rat).Mul(
		rawExecutionHours,
		big.NewRat(int64(100-projectBufferPercentage), 100),
	)
	return roundHoursToHalfHourMinutes(rawCommitmentHours), nil
}

func roundHoursToHalfHourMinutes(hours *big.Rat) *big.Rat {
	halfHours := new(big.Rat).Mul(hours, big.NewRat(2, 1))
	halfHours.Add(halfHours, big.NewRat(1, 2))
	roundedHalfHours := new(big.Int).Quo(halfHours.Num(), halfHours.Denom())
	return new(big.Rat).SetInt(
		new(big.Int).Mul(roundedHalfHours, big.NewInt(30)),
	)
}

func percentFactor(percentage *big.Rat) *big.Rat {
	return new(big.Rat).Sub(big.NewRat(1, 1), new(big.Rat).Quo(percentage, big.NewRat(100, 1)))
}

func DateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func DateKey(value time.Time) string { return DateOnly(value).Format("2006-01-02") }

func IsWeekend(value time.Time) bool {
	weekday := DateOnly(value).Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func CloneRat(value *big.Rat) *big.Rat {
	if value == nil {
		return new(big.Rat)
	}
	return new(big.Rat).Set(value)
}
