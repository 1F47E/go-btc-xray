package gui_buffer

import "container/list"

// Custom queue like data structure for the charts and logs
// Implements FIFO principle via linked list. New pushed to the back, old poped from the front.
// On every push copies all the data from the list to the flat array
// Data can be floats (charts) or strings (logs)
// Operations are read heavy. Writes 1 RPS, reads 10 RPS
type GuiBuffer struct {
	list *list.List
	size int
	data []box
	// flat copy of the data in diff formats to feed to the charts
	dataFlatFloat  []float64
	dataFlatString []string
}

// data container for the linked list
type box struct {
	data interface{} // float64 or string
}

func New(size int) *GuiBuffer {
	q := GuiBuffer{
		list: list.New(),
		size: size,
		data: make([]box, size),
	}
	return &q
}

func (q *GuiBuffer) AddNum(val int) {
	if val == 0 {
		return
	}
	f := float64(val)

	// wrap the data in a box
	q.add(box{data: f})

	// copy data over from data box to the flat array
	if q.dataFlatFloat == nil {
		q.dataFlatFloat = make([]float64, q.size)
	}
	for i, v := range q.data {
		// because q.data is preallocated we should skip all nil values
		// [_ _ _ _ _ _ _ X X] <- new data is pushed to the back
		if v.data == nil {
			continue
		}
		q.dataFlatFloat[i] = v.data.(float64)
	}
}

func (q *GuiBuffer) AddString(val string) {
	// wrap the data in a box
	q.add(box{data: val})

	// copy data over from data box to the flat array
	if q.dataFlatString == nil {
		q.dataFlatString = make([]string, q.size)
	}
	for i, v := range q.data {
		// skip nil boxes
		if v.data == nil {
			continue
		}
		q.dataFlatString[i] = v.data.(string)
	}
}

func (q *GuiBuffer) add(b box) {
	q.list.PushBack(b)
	if q.list.Len() > q.size {
		q.list.Remove(q.list.Front())
	}
	// update data
	// copy list elements to the slice
	// updateSlice(mirror, l, limit)
	// loop from back to front and update slice accordingly
	i := 0
	for e := q.list.Back(); e != nil; e = e.Prev() {
		if i >= q.size {
			break
		}
		idx := q.size - 1 - i
		if idx >= len(q.data) {
			break
		}
		q.data[idx] = e.Value.(box)
		i++
	}
}

// feed to the charts
func (q *GuiBuffer) GetFloats() []float64 {
	if q.dataFlatFloat == nil {
		q.dataFlatFloat = make([]float64, q.size)
	}
	return q.dataFlatFloat
}

// feed to the logs
func (q *GuiBuffer) GetStrings() []string {
	if q.dataFlatString == nil {
		q.dataFlatString = make([]string, q.size)
	}
	return q.dataFlatString
}

// for the info table
func (q *GuiBuffer) GetLastNum() int {
	if q.dataFlatFloat == nil || len(q.dataFlatFloat) == 0 {
		return 0
	}
	last := q.dataFlatFloat[len(q.dataFlatFloat)-1]
	return int(last)
}
