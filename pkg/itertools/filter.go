package itertools

import "iter"

func Filter[I any](in iter.Seq[I], predicate func(I) bool) iter.Seq[I] {
	return func(yield func(I) bool) {
		for i := range in {
			if predicate(i) {
				if !yield(i) {
					return
				}
			}
		}
	}
}
