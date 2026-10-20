package workspace

import "fmt"

type Phase struct {
	ID       string   `json:"id"`
	Projects []string `json:"projects"`
}
type Suggestion struct {
	Rule    string   `json:"rule"`
	Phases  []Phase  `json:"phases"`
	Cyclic  []string `json:"cyclic"`
	Blocked []string `json:"blocked"`
}

// SuggestPhases performs Kahn layering on internal declared relationships.
// No scope is silently excluded: this is a conservative ordering suggestion,
// not a determination of which dependencies require coordinated releases.
func SuggestPhases(facts Facts) Document {
	doc := NewDocument(facts)
	doc.Kind = "refactor-plan-suggestion"
	result := Suggestion{Rule: "all-internal-declared-dependencies", Phases: []Phase{}, Cyclic: []string{}, Blocked: []string{}}
	indegree := map[string]int{}
	consumers := map[string][]string{}
	for _, p := range doc.Facts.Projects {
		indegree[p.ID] = 0
	}
	for _, e := range doc.Facts.Dependencies {
		if e.Internal {
			indegree[e.Source]++
			consumers[e.Target] = append(consumers[e.Target], e.Source)
		}
	}
	ready := []string{}
	for _, p := range doc.Facts.Projects {
		if indegree[p.ID] == 0 {
			ready = append(ready, p.ID)
		}
	}
	for head := 0; head < len(ready); {
		end := len(ready)
		members := append([]string{}, ready[head:end]...)
		result.Phases = append(result.Phases, Phase{ID: fmt.Sprintf("phase-%d", len(result.Phases)+1), Projects: members})
		for ; head < end; head++ {
			for _, consumer := range consumers[ready[head]] {
				indegree[consumer]--
				if indegree[consumer] == 0 {
					ready = append(ready, consumer)
				}
			}
		}
	}
	for _, p := range doc.Facts.Projects {
		if indegree[p.ID] > 0 {
			result.Blocked = append(result.Blocked, p.ID)
		}
	}
	cyclic := cycleMembers(consumers, doc.Facts.Projects)
	for _, p := range doc.Facts.Projects {
		if cyclic[p.ID] {
			result.Cyclic = append(result.Cyclic, p.ID)
		}
	}
	doc.Suggestion = &result
	doc.Notes = append(doc.Notes,
		"Phases are suggestions starting at phase-1, not an upgrade strategy or a safety guarantee.",
		"Every observed internal edge constrains order, including test and unspecified scopes; external dependencies do not determine phases.",
		"Parent relationships, version compatibility, artifact publication and repository release units are not inferred from these two CSVs.",
		"Blocked includes cycle members and their downstream consumers; no steps are invented for them.")
	return doc
}

// Tarjan SCC distinguishes actual cycle members from downstream blocked nodes.
func cycleMembers(edges map[string][]string, projects []Project) map[string]bool {
	indices, low := map[string]int{}, map[string]int{}
	onStack, cyclic := map[string]bool{}, map[string]bool{}
	stack := []string{}
	next := 0
	var visit func(string)
	visit = func(id string) {
		next++
		indices[id] = next
		low[id] = next
		stack = append(stack, id)
		onStack[id] = true
		for _, target := range edges[id] {
			if indices[target] == 0 {
				visit(target)
				low[id] = min(low[id], low[target])
			} else if onStack[target] {
				low[id] = min(low[id], indices[target])
			}
		}
		if low[id] != indices[id] {
			return
		}
		component := []string{}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == id {
				break
			}
		}
		isCycle := len(component) > 1
		if !isCycle {
			for _, target := range edges[id] {
				if target == id {
					isCycle = true
				}
			}
		}
		if isCycle {
			for _, member := range component {
				cyclic[member] = true
			}
		}
	}
	for _, p := range projects {
		if indices[p.ID] == 0 {
			visit(p.ID)
		}
	}
	return cyclic
}
