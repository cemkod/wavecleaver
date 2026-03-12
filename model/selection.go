package model

// Selection tracks which cycles the user has selected.
type Selection struct {
	Selected map[int]bool // cycle index -> selected
	Last     int          // last clicked cycle index (for shift-click range)
}

func NewSelection() *Selection {
	return &Selection{
		Selected: make(map[int]bool),
		Last:     -1,
	}
}

// Toggle flips the selection state of a cycle.
func (s *Selection) Toggle(idx int) {
	if s.Selected[idx] {
		delete(s.Selected, idx)
	} else {
		s.Selected[idx] = true
	}
	s.Last = idx
}

// SelectRange selects all cycles between s.Last and idx (inclusive).
func (s *Selection) SelectRange(idx int) {
	if s.Last < 0 {
		s.Toggle(idx)
		return
	}
	lo, hi := s.Last, idx
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		s.Selected[i] = true
	}
	s.Last = idx
}

// Clear deselects all.
func (s *Selection) Clear() {
	s.Selected = make(map[int]bool)
	s.Last = -1
}

// IsSelected returns whether cycle idx is selected.
func (s *Selection) IsSelected(idx int) bool {
	return s.Selected[idx]
}

// SortedIndices returns selected cycle indices in ascending order.
func (s *Selection) SortedIndices() []int {
	indices := make([]int, 0, len(s.Selected))
	for i := range s.Selected {
		indices = append(indices, i)
	}
	// Simple insertion sort — selections are typically small.
	for i := 1; i < len(indices); i++ {
		for j := i; j > 0 && indices[j-1] > indices[j]; j-- {
			indices[j-1], indices[j] = indices[j], indices[j-1]
		}
	}
	return indices
}

// Count returns the number of selected cycles.
func (s *Selection) Count() int {
	return len(s.Selected)
}
