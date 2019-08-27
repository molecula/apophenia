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
	"fmt"
	"time"

	"github.com/molecula/apophenia"
)

// Card represents one card out of a series of decks; every 52 values
// represents one deck.
type Card uint16

var suits = []byte("CDHS")
var faces = []byte("A23456789TJQK")

func (c Card) String() string {
	if c >= 52 {
		return fmt.Sprintf("invalid-card-%04x", uint16(c))
	}
	return string([]byte{faces[c%13], suits[(c/13)%4]})
}

// Shuffle represents a particular permutation of a number of cards. The
// cards can be dealt out of it in order, once, with no repeats. (But if
// the shuffle had multiple decks, you can get cards which have the same
// face and suit.)
type Shuffle struct {
	dealt   int
	max     int
	shuffle *apophenia.Permutation
}

// NewShuffle yields a shuffle of one or more decks of cards.
func NewShuffle(decks int, seed int64) (*Shuffle, error) {
	if decks <= 0 || decks > 1000 {
		return nil, fmt.Errorf("number of decks (%d) must be positive, but under 1000", decks)
	}
	shuffle, err := apophenia.NewPermutation(52, 0, apophenia.NewSequence(seed))
	if err != nil {
		return nil, err
	}
	return &Shuffle{shuffle: shuffle, max: decks * 52}, nil
}

// Deal yields up to n cards from the shuffle, but stops when it runs out
// of cards.
func (s *Shuffle) Deal(n int) (c []Card, err error) {
	if n+s.dealt > s.max {
		err = fmt.Errorf("can't deal %d cards, only %d remaining", n, s.max-s.dealt)
		n = s.max - s.dealt
	}
	c = make([]Card, n)
	for i := 0; i < n; i++ {
		value := s.shuffle.Next()
		if int64(Card(value)) != value {
			err = fmt.Errorf("invalid card value %d -- too large", value)
		}
		c[i] = Card(value)
	}
	s.dealt += n
	return c, err
}

func main() {
	// This is not secure, don't do it for things where that matters.
	seed := time.Now().UnixNano()
	shuffle, err := NewShuffle(1, seed)
	if err != nil {
		fmt.Printf("fatal: couldn't create shuffle: %v", err)
		return
	}
	cards, err := shuffle.Deal(5)
	if err != nil {
		fmt.Printf("error dealing cards: %v\n", err)
		return
	}
	for _, c := range cards {
		fmt.Printf("  %s [%d]\n", c, c)
	}
}
