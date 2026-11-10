package engine

import "slices"

// Capabilities belong to an installed package version.
type Capabilities struct {
	Languages  []string `json:"languages"`
	Operations []string `json:"operations"`
	ScanFacts  []string `json:"scanFacts"`
}

func (c *Capabilities) Supports(operation string) bool {
	return c != nil && slices.Contains(c.Operations, operation)
}

func (c *Capabilities) SupportsFact(fact string) bool {
	return c.Supports("scan") && slices.Contains(c.ScanFacts, fact)
}
