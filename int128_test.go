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
	"testing"
)

func Test_Int128Rotate(t *testing.T) {
	cases := []struct {
		in         Uint128
		bits       uint64
		outL, outR Uint128
	}{
		{in: Uint128{Lo: 0x1}, bits: 1, outR: Uint128{Lo: 0x0, Hi: 1 << 63}, outL: Uint128{Lo: 0x2, Hi: 0}},
		{in: Uint128{Lo: 0x11}, bits: 4, outR: Uint128{Lo: 1, Hi: 1 << 60}, outL: Uint128{Lo: 0x110, Hi: 0}},
		{in: Uint128{Lo: 0x11}, bits: 65, outR: Uint128{Lo: 1 << 63, Hi: 8}, outL: Uint128{Lo: 0, Hi: 0x22}},
	}
	for _, c := range cases {
		u := c.in
		u.RotateRight(c.bits)
		if u != c.outR {
			t.Fatalf("rotate %s right by %d: expected %s, got %s",
				c.in, c.bits, c.outR, u)
		}
		u = c.in
		u.RotateLeft(c.bits)
		if u != c.outL {
			t.Fatalf("rotate %s left by %d: expected %s, got %s",
				c.in, c.bits, c.outL, u)
		}
	}
}

func Test_Int128Shift(t *testing.T) {
	cases := []struct {
		in         Uint128
		bits       uint64
		outL, outR Uint128
	}{
		{in: Uint128{Lo: 0x1}, bits: 1, outR: Uint128{Lo: 0x0, Hi: 0}, outL: Uint128{Lo: 0x2, Hi: 0}},
		{in: Uint128{Lo: 0x11}, bits: 4, outR: Uint128{Lo: 1, Hi: 0}, outL: Uint128{Lo: 0x110, Hi: 0}},
		{in: Uint128{Lo: 0x11, Hi: 0x3}, bits: 65, outR: Uint128{Lo: 1, Hi: 0}, outL: Uint128{Lo: 0, Hi: 0x22}},
		{in: Uint128{Lo: 0, Hi: 0x11}, bits: 68, outR: Uint128{Lo: 1, Hi: 0}, outL: Uint128{Lo: 0, Hi: 0}},
	}
	for _, c := range cases {
		u := c.in
		u.ShiftRight(c.bits)
		if u != c.outR {
			t.Fatalf("shift %s right by %d: expected %s, got %s",
				c.in, c.bits, c.outR, u)
		}
		u = c.in
		u.ShiftLeft(c.bits)
		if u != c.outL {
			t.Fatalf("shift %s left by %d: expected %s, got %s",
				c.in, c.bits, c.outL, u)
		}
	}
}
