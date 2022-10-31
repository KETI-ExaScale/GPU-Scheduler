package internal

import (
	"container/heap"
	"fmt"

	"k8s.io/client-go/tools/cache"
)

type KeyFunc func(obj interface{}) (string, error)

type heapItem struct {
	obj   interface{}
	index int
}

type itemKeyValue struct {
	key string
	obj interface{}
}

type data struct {
	items    map[string]*heapItem
	queue    []string //key of items
	keyFunc  KeyFunc
	lessFunc lessFunc
}

var (
	_ = heap.Interface(&data{})
)

func (h *data) Less(i, j int) bool {
	if i > h.Len() || j > h.Len() {
		return false
	}
	itemi, ok := h.items[h.queue[i]]
	if !ok {
		return false
	}
	itemj, ok := h.items[h.queue[j]]
	if !ok {
		return false
	}
	return h.lessFunc(itemi.obj, itemj.obj)
}

func (h *data) Len() int { return len(h.queue) }

func (h *data) Swap(i, j int) {
	h.queue[i], h.queue[j] = h.queue[j], h.queue[i]
	item := h.items[h.queue[i]]
	item.index = i
	item = h.items[h.queue[j]]
	item.index = j
}

func (h *data) Pop() interface{} {
	key := h.queue[len(h.queue)-1]
	h.queue = h.queue[0 : len(h.queue)-1]
	item, ok := h.items[key]
	if !ok {
		return nil
	}
	delete(h.items, key)
	return item.obj
}

func (h *data) Push(kv interface{}) {
	keyValue := kv.(*itemKeyValue)
	n := len(h.queue)
	h.items[keyValue.key] = &heapItem{keyValue.obj, n}
	h.queue = append(h.queue, keyValue.key)
}

func (h *data) Peek() interface{} {
	if len(h.queue) > 0 {
		return h.items[h.queue[0]].obj
	}
	return nil
}

type Heap struct {
	data *data
}

func (h *Heap) Add(obj interface{}) error {
	key, err := h.data.keyFunc(obj)
	if err != nil {
		return fmt.Errorf("Add>key error-%s", err)
	}
	if _, exists := h.data.items[key]; exists {
		h.data.items[key].obj = obj
		heap.Fix(h.data, h.data.items[key].index)
	} else {
		heap.Push(h.data, &itemKeyValue{key, obj})
	}
	return nil
}

func (h *Heap) AddIfNotPresent(obj interface{}) error {
	key, err := h.data.keyFunc(obj)
	if err != nil {
		return fmt.Errorf("AddIfNotPresent>key error - %s", err)
	}
	if _, exists := h.data.items[key]; !exists {
		heap.Push(h.data, &itemKeyValue{key, obj})
	}
	return nil
}

func (h *Heap) Update(obj interface{}) error {
	return h.Add(obj)
}

func (h *Heap) Delete(obj interface{}) error {
	key, err := h.data.keyFunc(obj)
	if err != nil {
		return fmt.Errorf("Delete>key error - %s", err)
	}
	if item, ok := h.data.items[key]; ok {
		heap.Remove(h.data, item.index)
		return nil
	}
	return fmt.Errorf("Delete>object not found")
}

func (h *Heap) Peek() interface{} {
	return h.data.Peek()
}

func (h *Heap) Pop() (interface{}, error) {
	obj := heap.Pop(h.data)
	if obj != nil {
		return obj, nil
	}
	return nil, fmt.Errorf("object was removed from heap data")
}

func (h *Heap) Get(obj interface{}) (interface{}, bool, error) {
	key, err := h.data.keyFunc(obj)
	if err != nil {
		return nil, false, cache.KeyError{Obj: obj, Err: err}
	}
	return h.GetByKey(key)
}

func (h *Heap) GetByKey(key string) (interface{}, bool, error) {
	item, exists := h.data.items[key]
	if !exists {
		return nil, false, nil
	}
	return item.obj, true, nil
}

func (h *Heap) List() []interface{} {
	list := make([]interface{}, 0, len(h.data.items))
	for _, item := range h.data.items {
		list = append(list, item.obj)
	}
	return list
}

func (h *Heap) Len() int {
	return len(h.data.queue)
}

func New(keyFn KeyFunc, lessFn lessFunc) *Heap {
	return NewWithRecorder(keyFn, lessFn)
}

func NewWithRecorder(keyFn KeyFunc, lessFn lessFunc) *Heap {
	return &Heap{
		data: &data{
			items:    map[string]*heapItem{},
			queue:    []string{},
			keyFunc:  keyFn,
			lessFunc: lessFn,
		},
	}
}

type lessFunc = func(item1, item2 interface{}) bool
