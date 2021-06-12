package internal

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testSegmentBuffer struct {
	curr int
	keys [][]Segment
}

func (l *testSegmentBuffer) Next() []Segment {
	defer func(l *testSegmentBuffer) {
		if l.curr == 0 {
			l.curr = len(l.keys) - 1
			return
		}
		l.curr--
	}(l)
	return l.keys[l.curr]
}

type testSegmentable []Segment

func (s testSegmentable) Segments() []Segment {
	return []Segment(s)
}

func (l *testSegmentBuffer) NextSegmentable() Segmentable {
	return testSegmentable(l.Next())
}

func generateSegments(tb testing.TB, prefix string, depth int) []Segment {
	tb.Helper()
	r := make([]Segment, depth)
	for i := 0; i < depth; i++ {
		r[i] = Segment(fmt.Sprintf("%v%v", prefix, i))
	}
	return r
}

func generateSegmentBuffer(tb testing.TB, depth, size int) *testSegmentBuffer {
	tb.Helper()
	l := &testSegmentBuffer{
		curr: size - 1,
		keys: make([][]Segment, size),
	}
	for i := 0; i < size; i++ {
		l.keys[i] = generateSegments(tb, fmt.Sprintf("%vp", i), depth)
	}
	return l
}

var cmpOpts = []cmp.Option{
	cmp.AllowUnexported(Node{}, LabelSet{}),
}

func TestLabelSetAccumulate(t *testing.T) {
	for tn, tc := range map[string]struct {
		add     []*Label
		want    LabelSet
		wantErr error
	}{
		"empty": {},
		"multiple unique labels": {
			add: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("bar"), "bar1"},
				{LabelKey("baz"), "baz1"},
			},
			want: LabelSet{
				seen: map[LabelKey]bool{
					"foo": true,
					"bar": true,
					"baz": true,
				},
				inOrder: []*Label{
					{LabelKey("bar"), "bar1"},
					{LabelKey("baz"), "baz1"},
					{LabelKey("foo"), "foo1"},
				},
			},
		},
		"non-unique label": {
			add: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("foo"), "foo2"},
				{LabelKey("bar"), "bar1"},
			},
			wantErr: ErrSeen,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			var got LabelSet
			err := got.Add(tc.add...)
			if err != nil && tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("LabelSet.Add(): got err %q is not %q", err, tc.wantErr)
				}
				return
			} else if err != nil || tc.wantErr != nil {
				t.Errorf("LabelSet.Add(): unexpected err value; got: %q, want: %q", err, tc.wantErr)
				return
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts...); diff != "" {
				t.Errorf("LabelSet.Add(): mismatch (-got, +want): %v", diff)
			}
		})
	}
}

func TestLabelSetAdd(t *testing.T) {
	for tn, tc := range map[string]struct {
		add       []*Label
		want      LabelSet
		errOnSeen bool
		wantErr   error
	}{
		"empty": {},
		"multiple unique labels": {
			add: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("bar"), "bar1"},
				{LabelKey("baz"), "baz1"},
			},
			want: LabelSet{
				seen: map[LabelKey]bool{
					"foo": true,
					"bar": true,
					"baz": true,
				},
				inOrder: []*Label{
					{LabelKey("bar"), "bar1"},
					{LabelKey("baz"), "baz1"},
					{LabelKey("foo"), "foo1"},
				},
			},
		},
		"non-unique label errors": {
			add: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("foo"), "foo2"},
				{LabelKey("bar"), "bar1"},
			},
			errOnSeen: true,
			wantErr:   ErrSeen,
		},
		"non-unique label continues": {
			add: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("foo"), "foo2"},
				{LabelKey("bar"), "bar1"},
			},
			want: LabelSet{
				seen: map[LabelKey]bool{
					"foo": true,
					"bar": true,
				},
				inOrder: []*Label{
					{LabelKey("bar"), "bar1"},
					{LabelKey("foo"), "foo1"},
				},
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			var got LabelSet
			err := got.add(tc.errOnSeen, tc.add...)
			if err != nil && tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("LabelSet.add(): got err %q is not %q", err, tc.wantErr)
				}
				return
			} else if err != nil || tc.wantErr != nil {
				t.Errorf("LabelSet.add(): unexpected err value; got: %q, want: %q", err, tc.wantErr)
				return
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts...); diff != "" {
				t.Errorf("LabelSet.add(): mismatch (-got, +want): %v", diff)
			}
		})
	}
}

type addMapCall struct {
	add  map[LabelKey]LabelValue
	want error
}

func TestLabelSetAddMap(t *testing.T) {
	for tn, tc := range map[string]struct {
		errOnSeen bool
		calls     []addMapCall
		want      LabelSet
	}{
		"empty": {},
		"multiple unique labels": {
			calls: []addMapCall{
				{
					add: map[LabelKey]LabelValue{
						"foo": "foo1",
						"bar": "bar1",
						"baz": "baz1",
					},
				},
			},
			want: LabelSet{
				seen: map[LabelKey]bool{
					"foo": true,
					"bar": true,
					"baz": true,
				},
				inOrder: []*Label{
					{LabelKey("bar"), "bar1"},
					{LabelKey("baz"), "baz1"},
					{LabelKey("foo"), "foo1"},
				},
			},
		},
		"non-unique label continues": {
			calls: []addMapCall{
				{
					add: map[LabelKey]LabelValue{
						"foo": "foo1",
						"bar": "bar1",
					},
				},
				{
					add: map[LabelKey]LabelValue{
						"foo": "foo2",
						"baz": "baz1",
					},
				},
			},
			want: LabelSet{
				seen: map[LabelKey]bool{
					"foo": true,
					"bar": true,
					"baz": true,
				},
				inOrder: []*Label{
					{LabelKey("bar"), "bar1"},
					{LabelKey("baz"), "baz1"},
					{LabelKey("foo"), "foo1"},
				},
			},
		},
		"non-unique label errors": {
			errOnSeen: true,
			calls: []addMapCall{
				{
					add: map[LabelKey]LabelValue{
						"foo": "foo1",
						"bar": "bar1",
					},
				},
				{
					add: map[LabelKey]LabelValue{
						"foo": "foo2",
						"baz": "baz1",
					},
					want: ErrSeen,
				},
			},
			want: LabelSet{
				seen: map[LabelKey]bool{
					"foo": true,
					"bar": true,
				},
				inOrder: []*Label{
					{LabelKey("bar"), "bar1"},
					{LabelKey("foo"), "foo1"},
				},
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			var got LabelSet
			for i, call := range tc.calls {
				t.Run(fmt.Sprintf("%v call %v", tn, i), func(t *testing.T) {
					err := got.addMap(tc.errOnSeen, call.add)
					if err != nil && call.want != nil {
						if !errors.Is(err, call.want) {
							t.Errorf("LabelSet.addMap(): got err %q is not %q", err, call.want)
						}
						return
					} else if err != nil || call.want != nil {
						t.Errorf("LabelSet.addMap(): unexpected err value; got: %q, want: %q", err, call.want)
						return
					}
				})
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts...); diff != "" {
				t.Errorf("LabelSet.addMap(): mismatch (-got, +want): %v", diff)
			}
		})
	}
}

func BenchmarkLabelSetAddBatch(b *testing.B) {
	for _, tc := range []int{
		1, 5, 10, 20, 50, 100,
	} {
		b.Run(fmt.Sprintf("Of%v", tc), func(b *testing.B) {
			s := generateSegments(b, "addBench", tc)
			labels := make([]*Label, len(s))
			for i := range s {
				labels[i] = &Label{
					Key:   LabelKey(s[i] + "k"),
					Value: LabelValue(s[i] + "v"),
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var ls LabelSet
				_ = ls.add(false, labels...)
			}
		})
	}
}

// BenchmarkLabelSetAddIndividual demonstrates why you should not call Add(...)
// in a loop.
func BenchmarkLabelSetAddIndividual(b *testing.B) {
	for _, tc := range []int{
		1, 5, 10, 20, 50, 100,
	} {
		b.Run(strconv.Itoa(tc), func(b *testing.B) {
			s := generateSegments(b, "addBench", tc)
			labels := make([]*Label, len(s))
			for i := range s {
				labels[i] = &Label{
					Key:   LabelKey(s[i] + "k"),
					Value: LabelValue(s[i] + "v"),
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var ls LabelSet
				for j := range labels {
					_ = ls.add(false, labels[j])
				}
			}
		})
	}
}

func BenchmarkLabelStringKey(b *testing.B) {
	for _, tc := range []int{
		1, 5, 10, 20, 50, 100,
	} {
		b.Run(strconv.Itoa(tc), func(b *testing.B) {
			s := generateSegments(b, "labelStringKeyBench", tc)
			labels := make([]*Label, len(s))
			for i := range s {
				labels[i] = &Label{
					Key:   LabelKey(s[i] + "k"),
					Value: LabelValue(s[i] + "v"),
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = labelStringKey(labels)
			}
		})
	}
}

func BenchmarkLabelSegments(b *testing.B) {
	for _, tc := range []int{
		1, 5, 10, 20, 50, 100,
	} {
		b.Run(fmt.Sprintf("%v", tc), func(b *testing.B) {
			s := generateSegments(b, "labelSegmentBench", tc)
			labels := make([]*Label, len(s))
			for i := range s {
				labels[i] = &Label{
					Key:   LabelKey(s[i] + "k"),
					Value: LabelValue(s[i] + "v"),
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = labelSegments(labels)
			}
		})
	}
}

func TestLabelStringKey(t *testing.T) {
	for tn, tc := range map[string]struct {
		want   string
		labels []*Label
	}{
		"empty": {},
		"single label": {
			want:   "foo=foo1;",
			labels: []*Label{{LabelKey("foo"), "foo1"}},
		},
		"multiple labels": {
			want: "foo=foo1;bar=bar1;baz=baz1;",
			labels: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("bar"), "bar1"},
				{LabelKey("baz"), "baz1"},
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			got := labelStringKey(tc.labels)
			if got != tc.want {
				t.Errorf("labelStringKey(): got: %q, want: %q", got, tc.want)
			}
		})
	}
}

func TestLabelSegments(t *testing.T) {
	for tn, tc := range map[string]struct {
		want   []Segment
		labels []*Label
	}{
		"empty": {},
		"single label": {
			want:   []Segment{"foo", "foo1"},
			labels: []*Label{{LabelKey("foo"), "foo1"}},
		},
		"multiple labels": {
			want: []Segment{"foo", "foo1", "bar", "bar1", "baz", "baz1"},
			labels: []*Label{
				{LabelKey("foo"), "foo1"},
				{LabelKey("bar"), "bar1"},
				{LabelKey("baz"), "baz1"},
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			got := labelSegments(tc.labels)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("labelSegments(): mismatch (-got, +want): %v", diff)
			}
		})
	}
}

func TestGet(t *testing.T) {
	testTrie := &Node{
		children: map[Segment]*Node{
			"foo": {
				children: map[Segment]*Node{
					"foo1": {
						value: "Hello, from foo1!",
						children: map[Segment]*Node{
							"bar": {
								children: map[Segment]*Node{
									"bar1": {
										value:    "Hello, from bar1!",
										children: map[Segment]*Node{},
									},
								},
							},
						},
					},
					"foo2": {
						children: map[Segment]*Node{
							"bar": {
								children: map[Segment]*Node{
									"bar2": {
										value:    "Hello, from bar2!",
										children: map[Segment]*Node{},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for tn, tc := range map[string]struct {
		key    []Segment
		want   []interface{}
		wantOk bool
	}{
		"empty": {},
		"existing prefix with intermediate values": {
			key:  []Segment{"foo", "foo1", "bar", "bar1"},
			want: []interface{}{"Hello, from foo1!", "Hello, from bar1!"},
		},
		"existing prefix with  full match value": {
			key:  []Segment{"foo", "foo2", "bar", "bar2"},
			want: []interface{}{"Hello, from bar2!"},
		},
		"existing prefix without value": {
			key: []Segment{"foo", "foo2"},
		},
		"non-existent prefix": {
			key: []Segment{"bleep", "bloop", "blorp"},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			got := Get(testTrie, tc.key)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Get(): mismatch (-got, +want): %v", diff)
			}
		})
	}
}

type insertCalls struct {
	want bool
	key  []Segment
}

func TestInsert(t *testing.T) {
	for tn, tc := range map[string]struct {
		wantNode *Node
		inserts  []insertCalls
	}{
		"empty key sets root node value": {
			wantNode: &Node{
				value:    "This is a test",
				children: map[Segment]*Node{},
			},
			inserts: []insertCalls{{want: true}},
		},
		"single insert": {
			wantNode: &Node{
				children: map[Segment]*Node{
					"foo": {
						children: map[Segment]*Node{
							"foo1": {
								children: map[Segment]*Node{
									"bar": {
										children: map[Segment]*Node{
											"bar2": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inserts: []insertCalls{
				{want: true, key: []Segment{"foo", "foo1", "bar", "bar2"}},
			},
		},
		"multiple non-overlapping inserts": {
			wantNode: &Node{
				children: map[Segment]*Node{
					"foo": {
						children: map[Segment]*Node{
							"foo1": {
								children: map[Segment]*Node{
									"bar": {
										children: map[Segment]*Node{
											"bar2": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
										},
									},
								},
							},
						},
					},
					"baz": {
						children: map[Segment]*Node{
							"baz2": {
								children: map[Segment]*Node{
									"quux": {
										children: map[Segment]*Node{
											"quux2": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inserts: []insertCalls{
				{want: true, key: []Segment{"foo", "foo1", "bar", "bar2"}},
				{want: true, key: []Segment{"baz", "baz2", "quux", "quux2"}},
			},
		},
		"common prefix equal length inserts": {
			wantNode: &Node{
				children: map[Segment]*Node{
					"foo": {
						children: map[Segment]*Node{
							"foo1": {
								children: map[Segment]*Node{
									"bar": {
										children: map[Segment]*Node{
											"bar1": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
											"bar2": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inserts: []insertCalls{
				{want: true, key: []Segment{"foo", "foo1", "bar", "bar1"}},
				{want: true, key: []Segment{"foo", "foo1", "bar", "bar2"}},
			},
		},
		"common prefix unequal length inserts": {
			wantNode: &Node{
				children: map[Segment]*Node{
					"foo": {
						children: map[Segment]*Node{
							"foo1": {
								children: map[Segment]*Node{
									"bar": {
										value: "This is a test",
										children: map[Segment]*Node{
											"bar1": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inserts: []insertCalls{
				{want: true, key: []Segment{"foo", "foo1", "bar", "bar1"}},
				{want: true, key: []Segment{"foo", "foo1", "bar"}},
			},
		},
		"subsequent calls overwrites value": {
			wantNode: &Node{
				children: map[Segment]*Node{
					"foo": {
						children: map[Segment]*Node{
							"foo1": {
								children: map[Segment]*Node{
									"bar": {
										children: map[Segment]*Node{
											"bar1": {
												value:    "This is a test",
												children: map[Segment]*Node{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inserts: []insertCalls{
				{want: true, key: []Segment{"foo", "foo1", "bar", "bar1"}},
				{want: false, key: []Segment{"foo", "foo1", "bar", "bar1"}},
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			testNode := NewNode()
			for i, call := range tc.inserts {
				t.Run(fmt.Sprintf("Insert() call %v", i), func(t *testing.T) {
					got := Insert(testNode, call.key, "This is a test")
					if got != call.want {
						t.Errorf("Insert(): got: %v, want: %v", got, call.want)
					}
				})
			}
			if diff := cmp.Diff(tc.wantNode, testNode, cmpOpts...); diff != "" {
				t.Errorf("Insert(): mismatch (-got, +want): %v", diff)
			}
		})
	}
}

func BenchmarkGetNoMatch(b *testing.B) {
	keys := generateSegmentBuffer(b, 10, 100)
	root := NewNode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get(root, keys.Next())
	}
}

func BenchmarkGetMatchEvery(b *testing.B) {
	keys := generateSegmentBuffer(b, 10, 100)
	root := NewNode()
	for _, key := range keys.keys {
		Insert(root, key, "This is a test")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get(root, keys.Next())
	}
}

func BenchmarkInsert(b *testing.B) {
	keys := generateSegmentBuffer(b, 10, 100)
	root := NewNode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Insert(root, keys.Next(), "This is a test")
	}
}

func BenchmarkTrieInsert(b *testing.B) {
	keys := generateSegmentBuffer(b, 10, 100)
	trie := Trie{
		root: NewNode(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Insert(keys.NextSegmentable(), "This is a test")
	}
}
