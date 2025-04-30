package stackx

type Stack[T any] struct {
	items []T
}

func New[T any]() *Stack[T] {
	return &Stack[T]{
		items: make([]T, 0),
	}
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}

	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, true
}

func (s *Stack[T]) SafePop() T {
	item, _ := s.Pop()
	return item
}

func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}

	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) SafePeek() T {
	item, _ := s.Peek()
	return item
}

func (s *Stack[T]) Size() int {
	return len(s.items)
}
