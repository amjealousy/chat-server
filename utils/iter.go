package utils

import (
	"context"
	"fmt"
	"iter"
)

func WithContext[T any](ctx context.Context, s iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		s(func(v T) bool {
			select {
			case <-ctx.Done():
				fmt.Println("[WithContext] Context cancelled")
				return false
			default:
				return yield(v)
			}
		})
	}
}
func TakeWhile[T any](s iter.Seq[T], pred func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		s(func(v T) bool {
			if !pred(v) {
				return false // Остановка!
			}
			return yield(v)
		})
	}
}

func SeqHistoryChan[T any](ch <-chan T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range ch {
			if !yield(v) {
				return
			}
		}
	}
}
