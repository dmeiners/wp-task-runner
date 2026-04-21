package task

// Task represents a job coming from any of the queues
type Task struct {
    Domain    string `json:"domain"`
    Slug      string `json:"slug,omitempty"`
    Source    string `json:"source,omitempty"`
    Version   string `json:"version,omitempty"`
    Action    string `json:"action"` // plugin-install, plugin-update, theme-install, theme-update, core-update
    Timestamp int64  `json:"timestamp,omitempty"`
    RequestID string `json:"request_id,omitempty"`
}
