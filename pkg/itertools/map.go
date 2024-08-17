package itertools

import "iter"

func Map[I, O any](in iter.Seq[I], f func(I) O) iter.Seq[O] {
	return func(yield func(O) bool) {
		for i := range in {
			if !yield(f(i)) {
				return
			}
		}
	}
}
