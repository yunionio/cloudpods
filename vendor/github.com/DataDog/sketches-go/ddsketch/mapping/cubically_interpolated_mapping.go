// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package mapping

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	enc "github.com/DataDog/sketches-go/ddsketch/encoding"
	"github.com/DataDog/sketches-go/ddsketch/pb/sketchpb"
)

const (
	A = 6.0 / 35.0
	B = -3.0 / 5.0
	C = 10.0 / 7.0
)

// A fast IndexMapping that approximates the memory-optimal LogarithmicMapping by extracting the floor value
// of the logarithm to the base 2 from the binary representations of floating-point values and cubically
// interpolating the logarithm in-between.
// More detailed documentation of this method can be found in:
// <a href="https://github.com/DataDog/sketches-java/">sketches-java</a>
type CubicallyInterpolatedMapping struct {
	relativeAccuracy      float64
	multiplier            float64
	normalizedIndexOffset float64
}

func NewCubicallyInterpolatedMapping(relativeAccuracy float64) (*CubicallyInterpolatedMapping, error) {
	if relativeAccuracy <= 0 || relativeAccuracy >= 1 {
		return nil, errors.New("The relative accuracy must be between 0 and 1.")
	}
	return &CubicallyInterpolatedMapping{
		relativeAccuracy: relativeAccuracy,
		multiplier:       7.0 / (10 * math.Log1p(2*relativeAccuracy/(1-relativeAccuracy))),
	}, nil
}

func NewCubicallyInterpolatedMappingWithGamma(gamma, indexOffset float64) (*CubicallyInterpolatedMapping, error) {
	if gamma <= 1 {
		return nil, errors.New("Gamma must be greater than 1.")
	}
	m := CubicallyInterpolatedMapping{
		relativeAccuracy: 1 - 2/(1+math.Exp(7.0/10*math.Log2(gamma))),
		multiplier:       1 / math.Log2(gamma),
	}
	m.normalizedIndexOffset = indexOffset - m.approximateLog(1)*m.multiplier
	return &m, nil
}

func (m *CubicallyInterpolatedMapping) Equals(other IndexMapping) bool {
	o, ok := other.(*CubicallyInterpolatedMapping)
	if !ok {
		return false
	}
	tol := 1e-12
	return (withinTolerance(m.multiplier, o.multiplier, tol) && withinTolerance(m.normalizedIndexOffset, o.normalizedIndexOffset, tol))
}

func (m *CubicallyInterpolatedMapping) Index(value float64) int {
	index := m.approximateLog(value)*m.multiplier + m.normalizedIndexOffset
	if index >= 0 {
		return int(index)
	} else {
		return int(index) - 1
	}
}

func (m *CubicallyInterpolatedMapping) Value(index int) float64 {
	return m.LowerBound(index) * (1 + m.relativeAccuracy)
}

func (m *CubicallyInterpolatedMapping) LowerBound(index int) float64 {
	return m.approximateInverseLog((float64(index) - m.normalizedIndexOffset) / m.multiplier)
}

// Return an approximation of log(1) + Math.log(x) / Math.log(base(2)).
func (m *CubicallyInterpolatedMapping) approximateLog(x float64) float64 {
	bits := math.Float64bits(x)
	e := getExponent(bits)
	s := getSignificandPlusOne(bits) - 1
	return ((A*s+B)*s+C)*s + e
}

// The exact inverse of approximateLog.
func (m *CubicallyInterpolatedMapping) approximateInverseLog(x float64) float64 {
	exponent := math.Floor(x)
	// Derived from Cardano's formula
	d0 := B*B - 3*A*C
	d1 := 2*B*B*B - 9*A*B*C - 27*A*A*(x-exponent)
	p := math.Cbrt((d1 - math.Sqrt(d1*d1-4*d0*d0*d0)) / 2)
	significandPlusOne := -(B+p+d0/p)/(3*A) + 1
	return buildFloat64(int(exponent), significandPlusOne)
}

func (m *CubicallyInterpolatedMapping) MinIndexableValue() float64 {
	return math.Max(
		math.Exp2((math.MinInt32-m.normalizedIndexOffset)/m.multiplier-m.approximateLog(1)+1), // so that index >= MinInt32:w
		minNormalFloat64*(1+m.relativeAccuracy)/(1-m.relativeAccuracy),
	)
}

func (m *CubicallyInterpolatedMapping) MaxIndexableValue() float64 {
	return math.Min(
		math.Exp2((math.MaxInt32-m.normalizedIndexOffset)/m.multiplier-m.approximateLog(float64(1))-1), // so that index <= MaxInt32
		math.Exp(expOverflow)/(1+m.relativeAccuracy),                                                   // so that math.Exp does not overflow
	)
}

func (m *CubicallyInterpolatedMapping) RelativeAccuracy() float64 {
	return m.relativeAccuracy
}

func (m *CubicallyInterpolatedMapping) gamma() float64 {
	return math.Exp2(1 / m.multiplier)
}

func (m *CubicallyInterpolatedMapping) ToProto() *sketchpb.IndexMapping {
	return &sketchpb.IndexMapping{
		Gamma:         m.gamma(),
		IndexOffset:   m.normalizedIndexOffset + m.approximateLog(1)*m.multiplier,
		Interpolation: sketchpb.IndexMapping_CUBIC,
	}
}

func (m *CubicallyInterpolatedMapping) Encode(b *[]byte) {
	enc.EncodeFlag(b, enc.FlagIndexMappingBaseCubic)
	enc.EncodeFloat64LE(b, m.gamma())
	enc.EncodeFloat64LE(b, m.normalizedIndexOffset)
}

func (m *CubicallyInterpolatedMapping) string() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("relativeAccuracy: %v, multiplier: %v, normalizedIndexOffset: %v\n", m.relativeAccuracy, m.multiplier, m.normalizedIndexOffset))
	return buffer.String()
}

var _ IndexMapping = (*CubicallyInterpolatedMapping)(nil)
