package workspace

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSuggestedPhasesAndBlocked(t *testing.T) {
	tests := []struct {
		name            string
		n               int
		edges           [][2]int
		phases          [][]string
		blocked, cyclic []string
	}{
		{"diamond", 7, [][2]int{{2, 1}, {3, 1}, {4, 2}, {4, 3}, {5, 1}, {6, 4}, {7, 4}, {7, 6}}, [][]string{{"1"}, {"2", "3", "5"}, {"4"}, {"6"}, {"7"}}, []string{}, []string{}},
		{"isolated", 3, nil, [][]string{{"1", "2", "3"}}, []string{}, []string{}},
		{"cycle downstream", 5, [][2]int{{2, 3}, {3, 2}, {4, 3}, {5, 1}}, [][]string{{"1"}, {"5"}}, []string{"2", "3", "4"}, []string{"2", "3"}},
		{"self", 2, [][2]int{{1, 1}}, [][]string{{"2"}}, []string{"1"}, []string{"1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := func(i int) string { return string(rune('0' + i)) }
			facts := Facts{Projects: []Project{}, Dependencies: []Dependency{}}
			for i := 1; i <= test.n; i++ {
				facts.Projects = append(facts.Projects, Project{ID: id(i)})
			}
			for _, edge := range test.edges {
				facts.Dependencies = append(facts.Dependencies, Dependency{Source: id(edge[0]), Target: id(edge[1]), Internal: true, Scopes: []string{"test"}})
			}
			// External references remain facts but cannot block local phases.
			facts.Dependencies = append(facts.Dependencies, Dependency{Source: "1", Target: "external", Internal: false})
			got := SuggestPhases(facts).Suggestion
			phases := [][]string{}
			for _, p := range got.Phases {
				phases = append(phases, p.Projects)
			}
			if !reflect.DeepEqual(phases, test.phases) || !reflect.DeepEqual(got.Blocked, test.blocked) || !reflect.DeepEqual(got.Cyclic, test.cyclic) {
				t.Fatalf("unexpected suggestion: %+v", got)
			}
		})
	}
}
func TestInputOrderDoesNotChangeDocument(t *testing.T) {
	a := Facts{Projects: []Project{{ID: "b"}, {ID: "a"}, {ID: "c"}}, Dependencies: []Dependency{{Source: "b", Target: "a", Internal: true}, {Source: "c", Target: "a", Internal: true}}}
	b := Facts{Projects: []Project{{ID: "c"}, {ID: "a"}, {ID: "b"}}, Dependencies: []Dependency{a.Dependencies[1], a.Dependencies[0]}}
	x, _ := json.Marshal(SuggestPhases(a))
	y, _ := json.Marshal(SuggestPhases(b))
	if string(x) != string(y) {
		t.Fatal("unstable output")
	}
}
