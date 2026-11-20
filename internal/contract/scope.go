package contract

// ScanScope describes selection, not code ownership or semantic completeness.
type ScanScope struct {
	SchemaVersion   int      `json:"schemaVersion"`
	Config          string   `json:"config"`
	ConfigSHA256    string   `json:"configSha256"`
	Base            string   `json:"base"`
	Include         []string `json:"include"`
	Exclude         []string `json:"exclude"`
	Considered      int      `json:"considered"`
	Included        int      `json:"included"`
	Excluded        int      `json:"excluded"`
	SelectionSHA256 string   `json:"selectionSha256"`
}
type FileSelectionRef struct {
	FileCount int    `json:"fileCount"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
}
type FileSelectionReceipt struct {
	SHA256    string `json:"sha256"`
	FileCount int    `json:"fileCount"`
}
type ComparisonScope struct {
	Policy string     `json:"policy"`
	Before *ScanScope `json:"before,omitempty"`
	After  *ScanScope `json:"after,omitempty"`
}
