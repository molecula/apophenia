// Copyright 2019 Pilosa Corp.
//
// Licensed under the BSD 3-Clause license (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     https://opensource.org/licenses/BSD-3-Clause
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apophenia

import (
	"fmt"
	"math"
)

// Zipf produces a series of values following a Zipf distribution.
// It is initialized with values q, v, and max, and produces values
// in the range [0,max) such that the probability of a value k is
// proportional to (v+k) ** -q. The input value v must be >= 1, and
// q must be > 1.
//
// This is based on the same paper used for the golang stdlib Zipf
// distribution:
//
// "Rejection-Inversion to Generate Variates
// from Monotone Discrete Distributions"
// W.Hormann, G.Derflinger [1996]
// http://eeyore.wu-wien.ac.at/papers/96-04-04.wh-der.ps.gz
//
// This implementation differs from stdlib's in that it is seekable; you
// can get the Nth value in a theoretical series of results in constant
// time, without having to generate the whole series linearly, and that
// we used the names q and v rather than s and v.
type Zipf struct {
	src                  Sequence
	seed                 uint32
	q                    float64
	v                    float64
	max                  float64
	oneMinusQ            float64
	oneOverOneMinusQ     float64
	hImaxOneHalf         float64
	hX0MinusHImaxOneHalf float64 // hX0 is only ever used as hX0 - h(i[max] + 1/2)
	s                    float64
	idx                  uint64
}

// Helper functions from the original algorithm. These are slightly too
// expensive to inline, but also inlining them (by hand) produced zero
// observable gain.
func h(z *Zipf, x float64) float64 {
	return math.Exp(z.oneMinusQ*math.Log(z.v+x)) * z.oneOverOneMinusQ
}

func hInv(z *Zipf, x float64) float64 {
	return -z.v + math.Exp(z.oneOverOneMinusQ*math.Log(z.oneMinusQ*x))
}

// NewZipf returns a new Zipf object with the specified q, v, and
// max, and with its random source seeded in some way by seed.
// The sequence of values returned is consistent for a given set
// of inputs. The seed parameter can select one of multiple sub-sequences
// of the given sequence.
func NewZipf(q float64, v float64, max uint64, seed uint32, src Sequence) (z *Zipf, err error) {
	if math.IsNaN(q) || math.IsNaN(v) {
		return nil, fmt.Errorf("q (%g) and v (%g) must not be NaN for Zipf distribution", q, v)
	}
	if q <= 1 || v < 1 {
		return nil, fmt.Errorf("need q > 1 (got %g) and v >= 1 (got %g) for Zipf distribution", q, v)
	}
	if src == nil {
		return nil, fmt.Errorf("need a usable PRNG apophenia.Sequence")
	}
	oneMinusQ := 1 - q
	oneOverOneMinusQ := 1 / (1 - q)
	z = &Zipf{
		q:                q,
		v:                v,
		max:              float64(max),
		seed:             seed,
		oneMinusQ:        oneMinusQ,
		oneOverOneMinusQ: oneOverOneMinusQ,
		idx:              0,
	}
	hX0 := h(z, 0.5) - math.Exp(math.Log(v)*-q)
	z.hImaxOneHalf = h(z, z.max+0.5)
	z.hX0MinusHImaxOneHalf = hX0 - z.hImaxOneHalf
	z.s = 1 - hInv(z, h(z, 1.5)-math.Exp(math.Log(v+1)*-q))
	z.src = src
	return z, nil
}

// Nth returns the Nth value from the sequence associated with the
// given Zipf. The value is fully determined by the input values
// (q, v, max, and seed) and the index.
func (z *Zipf) Nth(index uint64) uint64 {
	z.idx = index
	offset := OffsetFor(SequenceZipfU, z.seed, 0, index)
	for {
		bits := z.src.BitsAt(offset)
		uInt := bits.Lo
		u := float64(uInt&(1<<53-1)) / (1 << 53)
		u = z.hImaxOneHalf + u*z.hX0MinusHImaxOneHalf
		x := hInv(z, u)
		k := math.Floor(x + 0.5)
		if k-x <= z.s {
			return uint64(k)
		}
		if u >= h(z, k+0.5)-math.Exp(-math.Log(z.v+k)*z.q) {
			return uint64(k)
		}
		// the low-order 24 bits of the high-order 64-bit word
		// are the "iteration", which started as zero. Assuming we
		// don't need more than ~16.7M values, we're good. The expected
		// average is about 1.1.
		offset.Hi++
	}
}

// Next returns the "next" value -- the one after the last one requested, or
// value 1 if none have been requested before.
func (z *Zipf) Next() uint64 {
	return z.Nth(z.idx + 1)
}
