# Apophenia -- seeking patterns in randomness

Apophenia provides an approximate emulation of a seekable pseudo-random
number generator. You provide a seed, and get a generator which can generate
a large number of pseudo-random bits which will occur in a predictable
pattern, but you can seek anywhere in that pattern in constant time.

Apophenia's interface is intended to be similar to that of stdlib's
`math/rand`; the `Sequence` interface type is a strict superset
of `rand.Source64`. However, you can use the sequence's `Seek` method
to control what the corresponding source will generate next. (In
principle, you could implement other distinct sequence types, but the
AES-128 implementation is the only one right now.)

If you want something similar to this, but intended for cryptographic
purposes, look at Fortuna, which is similar in some ways, but
intended to be usable in cryptographic contexts:
	https://www.schneier.com/academic/fortuna/

The original use case for apophenia was an internal tool which generates
data sets, where we wanted to be able to generate the same resulting set
of data, while generating the data in different orders; the simplest way
to do that is to ensure that, for a given position in a matrix, you are
generating the same random bits when you generate that position, regardless
of which other positions have already been generated, or in what order.

## Implementation Notes

A PRNG should generate a sequence of unpredictable bits, with high entropy,
and minimal ability to predict future bits from previous bits.

If you view the input to AES-128 as a 128-bit number, the outputs for
sequential numbers are unpredictable -- knowing the current number or
its hash result gives you no information about the bits in the hash
result of the next number. So AES-128 plus a sequence of 128-bit values
ranging from 0 to (1<<128)-1 is *just like* a PRNG. But you can seek
to any location within its stream and generate the bits at that location
in constant time, because you never need to know about a previous "state".

So AES-128, seeded, is equivalent to a PRNG which generates random
bits with a period of (1<<135).

This design may have serious fundamental flaws, but it worked out in
light testing and I'm an optimist. Most obviously, if you generate a mere 2^64
consecutive blocks that are genuinely random, the expected number of collisions
is about 1; the expected number of collisions with apophenia is exactly 0.

### Sequences and Offsets

Apophenia's underlying implementation admits 128-bit keys, and 128-bit
offsets within each sequence. In most cases:

* That's more space than we need.
* Working with a non-native type for item numbers is annoying,
  but 64 bits is enough range.
* It would be nice to avoid using the *same* pseudo-random values
  for different things.
* Even when those things have the same basic identifying ID or
  value.

For instance, say you wish to generate a few billion objects, each of
which has multiple associated values, and you want to generate the values
randomly. Some values might follow a Zipf distribution, others might just be
"U % N" for some N. If you use the item number as a key, and seek to the same
position for each of these, and get the same bits for each of these, you may
get unintended similarities or correlations between them.

With this in mind, apophenia divides its 128-bit offset space into a number
of spaces. 8 of the bits are used for a sequence-type value, one of:

* `SequenceDefault`
* `SequencePermutationK`/`SequencePermutationF`: permutations
* `SequenceWeighted`: weighted bits
* `SequenceLinear`: linear values within a range
* `SequenceZipfU`: uniforms to use for Zipf values
* `SequenceRandSource`: default offsets for the rand.Source
* `SequenceUser1`/`SequenceUser2`: reserved for non-apophenia usage

Other values are not yet defined, but are reserved.

Within most of these spaces, the rest of the high word of the offset is used
for a 'seed' (used to select different sequences) and an 'iteration' (used
for successive values consumed by an algorithm). For instance, the Zipf
implementation, much like the Go stdlib implementation, may consume
multiple values; it does so by requesting blocks with consecutive iteration
values.

The low-order word is treated as a 64-bit item ID.

```
High-order word:
0-23                     24-31   32-63
[iteration              ][seq   ][seed                         ]

Low-order word:
0-63
[id                                                            ]
```

The convenience function `OffsetFor(sequence, seed, iteration, id)`
supports this usage.

As a side effect, if generating additional values for a given seed and
id, you can increment the high-order word of the `Uint128`,
and if generating values for a new id, you can increment the low-order
word. If your algorithm consumes more than 2^24 values for a single
operation, incrementing will bump the sequence value -- meaning you start
generating values that were used for iterations of a different kind
of sequence entirely, not values used for a different seed of this
sequence.

This division of space is arbitrary, and you're welcome to ignore it
and use the space however you like.

### Permutations

Apophenia provides a permutation generator, which generates the values
from 0 through N in an arbitrary order. For even small N, the number of
theoretically possible permutations vastly exceeds the space of possible
seed values, so only a vanishingly small number of possible permutations
will ever be generated (around 2^64 for any given apophenia seed), but
in practice that's still quite a lot.

### Zipf Distribution

Apophenia provides a seekable Zipf generator. This is basically equivalent
to using stdlib's Zipf type with an `apophenia.Sequence` as the `rand.Source`
backing its `rand.Rand`, only you don't have to mess around with it as much
to control that. Also, this one uses `q` and `v` rather than `s` and `v` as
the parameter names to align with the paper it was derived from.

#### Iteration usage

For the built-in consumers:

* Weighted consumes log2(scale) iterated values.
* Zipf consumes an *average* of no more than about 1.1 values.
* Permutation consumes one iterated value per 128 rounds of permutation,
  where rounds is equal to `6*ceil(log2(max))`. (For instance, a second
  value is consumed around a maximum of 2^22, and a third around 2^43.)
* Nothing else uses more than one iterated value.
