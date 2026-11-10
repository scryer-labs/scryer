package engine

import "slices"

// Capabilities belong to an installed package version.
type Capabilities struct {
	ResolveFacts        []string `json:"resolveFacts,omitempty"`
	ResolveDependencies bool     `json:"resolveDependencies,omitempty"`
	Languages           []string `json:"languages"`
	Operations          []string `json:"operations"`
	ScanFacts           []string `json:"scanFacts"`
}

func (c *Capabilities) Supports(operation string) bool {
	return c != nil && slices.Contains(c.Operations, operation)
}

func (c *Capabilities) SupportsFact(fact string) bool {
	return c.Supports("scan") && slices.Contains(c.ScanFacts, fact)
}

func (c *Capabilities) ResolvesFact(fact string) bool {
	return c.SupportsFact(fact) && (slices.Contains(c.ResolveFacts, fact) || (fact == "dependencies" && c.ResolveDependencies))
}
