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
	"errors"
	"math/bits"
)

// PermutationGenerator provides a way to pass integer IDs through a permutation
// map that is pseudorandom but repeatable. This could be done with `rand.Perm`,
// but that would require storing a slice of `[Items]int64`, which we want to
// avoid for large values of Items.
//
// Not actually cryptographically secure.
//
// This implementation is based on: http://arxiv.org/abs/1208.1176v2
//
// This simulates the results of a shuffle in a way allowing a lookup of
// the results of the shuffle for any given position, in time proportional
// to a number of "rounds", each of which is 50% likely to swap a slot
// with another slot. The number of rounds needed to achieve a reasonable
// probability of safety is `log(N)*6` or so.
//
// Each permutation is fully defined by a "key", consisting of:
//   1. A key "KF" naming a value in [0,max) for each round.
//   2. A series of round functions mapping values in [0,max) to bits,
//      one for each round.
// I refer to these as K[r] and F[r]. Thus, K[0] is the index used to
// compute swap operations four round 0, and F[0] is the series of bits
// used to determine whether a swap is performed, with F[0][0] being
// the swap decision for slot 0 in round 0. (Except it probably isn't,
// because the swap decision is actually made based on the highest index
// in a pair, to ensure that a swap between A and B always uses the same
// decision bit.)
//
// K values are generated using the SequencePermutationK range of offsets,
// with the provided seed, and iteration id 0.
//
// F values are generated using the SequencePermutationF range of offsets,
// with the provided seed, and iteration ID corresponding to the current
// round.
//
// For F values, we set byte 8 of the plain text to 0x00, and use
// encoding/binary to dump the slot number into the first 8 bytes. This
// yields 128 values, which we treat as the values for the first 128 rounds,
// and then recycle for rounds 129+ if those exist. This is not very
// secure, but we're already at 1/2^128 chances by that time and don't care.
// We could probably trim rounds to 64 or so and not lose much data.
type Permutation struct {
	src      Sequence
	permSeed uint32
	max      int64
	counter  int64
	rounds   int
	bits     Uint128
	k        []uint64
}

// NewPermutation creates a Permutation which generates values in [0,max),
// from a given Sequence and seed value.
//
// The seed parameter selects different shuffles, and is useful if you need
// to generate multiple distinct shuffles from the same underlying sequence.
// Treat it as a secondary seed.
func NewPermutation(max int64, seed uint32, src Sequence) (*Permutation, error) {
	if max < 1 {
		return nil, errors.New("period must be positive")
	}
	// number of rounds to get "good" results is roughly 6 log N.
	bits := 64 - bits.LeadingZeros64(uint64(max))
	p := Permutation{max: max, rounds: 6 * bits, counter: 0}

	p.src = src
	p.k = make([]uint64, p.rounds)
	p.permSeed = seed
	// Naive modulo arithmetic gives a slight bias towards the low
	// end of the range. Let's avoid that.
	maxMultiple := (^uint64(0) / uint64(p.max)) * uint64(p.max)
	for i := uint64(0); i < uint64(p.rounds); i++ {
		offset := OffsetFor(SequencePermutationK, p.permSeed, 0, i)
		bits := p.src.BitsAt(offset)
		// Skip things outside this range, so the range of
		// accepted values is an even multiple of p.max, so
		// all values in the range are equally likely.
		for bits.Lo >= maxMultiple {
			offset.Hi++
			bits = p.src.BitsAt(offset)
		}
		p.k[i] = p.src.BitsAt(offset).Lo % uint64(p.max)
	}
	return &p, nil
}

// Next generates the next value from the permutation.
func (p *Permutation) Next() (ret int64) {
	return p.nextValue()
}

// Nth generates the Nth value from the permutation. For instance,
// given a new permutation, calling Next once produces the same
// value you'd get from calling Nth(0). Seeking using Nth changes
// the offset that Next counts from; after calling Nth(x), you
// would get the same result from Next() that you would from Nth(x+1).
//
// Negative offsets count from the end; if p.max is 10, then Nth(9) and
// Nth(-1) produce the same value.
func (p *Permutation) Nth(n int64) (ret int64) {
	if n < 0 {
		n = p.max + (n % p.max)
	}
	p.counter = n
	ret = p.nextValue()
	return ret
}

func (p *Permutation) nextValue() int64 {
	p.counter = int64(uint64(p.counter) % uint64(p.max))
	x := uint64(p.counter)
	p.counter++
	// a value which can't possibly be the next value we need, so we
	// always hash on the first pass.
	prev := uint64(p.max) + 1
	offset := OffsetFor(SequencePermutationF, p.permSeed, 0, 0)
	for i := uint64(0); i < uint64(p.rounds); i++ {
		if i > 0 && i&127 == 0 {
			offset.Hi++
			// force regeneration of bits down below
			prev = uint64(p.max) + 1
		}
		xPrime := (p.k[i] + uint64(p.max) - x) % uint64(p.max)
		xCaret := x
		if xPrime > xCaret {
			xCaret = xPrime
		}
		if xCaret != prev {
			offset.Lo = xCaret
			p.bits = p.src.BitsAt(offset)
			prev = xCaret
		}
		if p.bits.Bit(i) != 0 {
			x = xPrime
		}
	}
	return int64(x)
}
