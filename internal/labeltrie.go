package internal

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type LabelKey string

type LabelValue string

type Label struct {
	Key   LabelKey
	Value LabelValue
}

type LabelSet struct {
	inOrder []*Label
	seen    map[LabelKey]bool
}

func copySeen(src map[LabelKey]bool) map[LabelKey]bool {
	dst := make(map[LabelKey]bool)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

var ErrSeen = errors.New("label with key already seen")

func (s *LabelSet) add(errOnSeen bool, labels ...*Label) error {
	nl := len(labels)
	if nl == 0 {
		return nil
	}
	if s.seen == nil {
		s.seen = make(map[LabelKey]bool)
	}
	local := make([]*Label, nl)
	var i int
	for _, l := range labels {
		if s.seen[l.Key] {
			if errOnSeen {
				return fmt.Errorf("%w: %v", ErrSeen, l.Key)
			}
			continue
		}
		local[i] = l
		s.seen[l.Key] = true
		i++
	}
	s.inOrder = append(s.inOrder, local[:i]...)
	sort.Slice(s.inOrder, func(i, j int) bool {
		return s.inOrder[i].Key < s.inOrder[j].Key
	})
	return nil
}

func (s *LabelSet) Accumulate(labels ...*Label) {
	_ = s.add(false, labels...)
}

func (s *LabelSet) Add(labels ...*Label) error {
	return s.add(true, labels...)
}

func (s *LabelSet) addMap(errOnSeen bool, labels map[LabelKey]LabelValue) error {
	nl := len(labels)
	if nl == 0 {
		return nil
	}
	if s.seen == nil {
		s.seen = make(map[LabelKey]bool)
	}
	local := make([]*Label, nl)
	var i int
	for k, v := range labels {
		if s.seen[k] {
			if errOnSeen {
				return fmt.Errorf("%w: %v", ErrSeen, k)
			}
			continue
		}
		local[i] = &Label{k, v}
		s.seen[k] = true
		i++
	}
	s.inOrder = append(s.inOrder, local[:i]...)
	sort.Slice(s.inOrder, func(i, j int) bool {
		return s.inOrder[i].Key < s.inOrder[j].Key
	})
	return nil
}

func (s *LabelSet) AccumulateMap(labels map[LabelKey]LabelValue) {
	_ = s.addMap(false, labels)
}

func (s *LabelSet) AddMap(labels map[LabelKey]LabelValue) error {
	return s.addMap(true, labels)
}

func (s *LabelSet) Copy() LabelSet {
	return LabelSet{
		inOrder: s.inOrder[:],
		seen:    copySeen(s.seen),
	}
}

func labelStringKey(labels []*Label) string {
	var sb strings.Builder
	for _, l := range labels {
		_, _ = sb.Write([]byte(l.Key))
		_, _ = sb.WriteRune('=')
		_, _ = sb.Write([]byte(l.Value))
		_, _ = sb.WriteRune(';')
	}
	return sb.String()
}

func (s LabelSet) String() string {
	return labelStringKey(s.inOrder)
}

func labelSegments(labels []*Label) []Segment {
	l := len(labels)
	if l == 0 {
		return nil
	}
	segs := make([]Segment, 2*l)
	for i := range labels {
		segs[(2 * i)] = Segment(labels[i].Key)
		segs[(2*i)+1] = Segment(labels[i].Value)
	}
	return segs
}

func (s LabelSet) Segments() []Segment {
	return labelSegments(s.inOrder)
}

type Segment string

type Segmentable interface {
	Segments() []Segment
}

type Node struct {
	value    interface{}
	children map[Segment]*Node
}

func NewNode() *Node {
	return &Node{
		children: make(map[Segment]*Node),
	}
}

// Insert the value with the provided key. Returns true if the operation has
// set a new value, and false if a value was overwritten for an existing key.
func Insert(node *Node, key []Segment, value interface{}) bool {
	for _, seg := range key {
		child, has := node.children[seg]
		if !has {
			child = NewNode()
			node.children[seg] = child
		}
		node = child
	}
	novel := node.value == nil
	node.value = value
	return novel
}

// Get all values matching the provided key.
func Get(node *Node, key []Segment) (values []interface{}) {
	for _, seg := range key {
		child, has := node.children[seg]
		if !has {
			return
		}
		node = child
		if node.value != nil {
			values = append(values, node.value)
		}
	}
	return
}

type Trie struct {
	root *Node
}

func (t *Trie) Insert(key Segmentable, value interface{}) bool {
	return Insert(t.root, key.Segments(), value)
}

func (t *Trie) Get(key Segmentable) []interface{} {
	return Get(t.root, key.Segments())
}

func NewTrie() *Trie {
	return &Trie{
		root: NewNode(),
	}
}
