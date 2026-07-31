package xifer

import "sync"

type MailBox[T any] struct {
	limit      int
	chHasItems chan struct{}

	mu    sync.RWMutex
	items []T
}

func newMailBox[T any](limit int) *MailBox[T] {
	return &MailBox[T]{
		limit:      limit,
		chHasItems: make(chan struct{}, 1),
		items:      make([]T, 0),
	}
}

func (m *MailBox[T]) Notify() <-chan struct{} {
	return m.chHasItems
}

func (m *MailBox[T]) Items() []T {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := make([]T, len(m.items))
	copy(items, m.items)

	m.items = make([]T, 0, len(items))

	return items
}

func (m *MailBox[T]) add(item T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.items) == m.limit {
		// drop the front item and add the new
		m.items = append(m.items[1:], item)
	} else {
		m.items = append(m.items, item)
	}

	// only send a notification if chHasItems has space
	select {
	case m.chHasItems <- struct{}{}:
	default:
	}
}
