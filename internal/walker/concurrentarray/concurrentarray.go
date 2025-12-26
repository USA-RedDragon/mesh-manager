package concurrentarray

import "sync"

type ConcurrentArray[T comparable] struct {
	data []T
	mu   sync.Mutex
}

func (ca *ConcurrentArray[T]) Contains(value T) bool {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	for _, item := range ca.data {
		if item == value {
			return true
		}
	}
	return false
}

// ContainsOrSet returns true if the value is already in the array, otherwise it adds the value to the array and returns false
func (ca *ConcurrentArray[T]) ContainsOrSet(value T) bool {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	for _, item := range ca.data {
		if item == value {
			return true
		}
	}
	ca.data = append(ca.data, value)
	return false
}

func (ca *ConcurrentArray[T]) Append(value T) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.data = append(ca.data, value)
}

func (ca *ConcurrentArray[T]) Remove(value T) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	for i, item := range ca.data {
		if item == value {
			ca.data = append(ca.data[:i], ca.data[i+1:]...)
			return
		}
	}
}

func (ca *ConcurrentArray[T]) Len() int {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	return len(ca.data)
}
