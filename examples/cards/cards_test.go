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

package main

import (
	"strings"
	"testing"
	"time"
)

func Test_CardNoDuplicates(t *testing.T) {
	seen := make([]uint64, 1024)
	seed := time.Now().UnixNano()
	for i := 0; i < 10; i++ {
		decks := 23 + i
		shuffle, err := NewShuffle(decks, seed)
		if err != nil {
			t.Fatalf("error generating shuffle of %d decks: %v", decks, err)
		}
		// deal 51 cards decks times, leaving decks cards to deal
		for j := 0; j < decks; j++ {
			dealt, err := shuffle.Deal(51)
			if err != nil {
				t.Fatalf("unexpected error from deal: %v", err)
			}
			for _, c := range dealt {
				mask := uint64(1 << (c & 63))
				word := c >> 6
				if seen[word]&mask != 0 {
					t.Fatalf("deal %d: saw card %s [%d] again\n", j, c, c)
				}
			}
		}
		dealt, err := shuffle.Deal(decks + 1)
		if err == nil {
			t.Fatalf("expected error not generated dealing too many cards")
		}
		if !strings.HasPrefix(err.Error(), "can't deal ") {
			t.Fatalf("unexpected error, expected \"can't deal\" message, got %v", err)
		}
		if len(dealt) != decks {
			t.Fatalf("expected exactly %d remaining cards, got %d", decks, len(dealt))
		}
	}
}
