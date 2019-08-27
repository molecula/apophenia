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
	"testing"
)

func Benchmark_WeightedDistribution(b *testing.B) {
	src := NewSequence(0)
	w, err := NewWeighted(src)
	if err != nil {
		b.Fatalf("couldn't make weighted: %v", err)
	}
	scales := []uint64{3, 6, 12, 18, 24, 63}
	for _, scale := range scales {
		off := OffsetFor(SequenceWeighted, 0, 0, 0)
		scaled := uint64(1 << scale)
		b.Run(fmt.Sprintf("Scale%d", scale), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				w.Bits(off, 1, scaled)
				w.Bits(off, scaled/2, scaled)
				w.Bits(off, scaled-1, scaled)
			}
		})
	}

}
