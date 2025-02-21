package monitors

import (
	"database/sql"
	"github.com/google/uuid"
)

// AgentInfo represents the complete metrics information for an agent
type AgentInfo struct {
	Version    string       `json:"version"`
	AgentID    string       `json:"agent_id"`
	IPAddress  string       `json:"IPAddress"`
	Name       string       `json:"name"`
	OS         string       `json:"os"`
	LastSync   sql.NullTime `json:"lastSync"`
	SDKVersion string       `json:"SDKVersion"`
	Metrics    []Metric
	Disks      []DiskMetric
}

// AgentMetricResponse represents the complete metrics data for an agent
type AgentMetricResponse struct {
	Status     string     `json:"status"`
	SystemInfo SystemInfo `json:"systemInfo"`
	Uptime     string     `json:"uptime"`
	AgentInfo  AgentInfo  `json:"agent_info"`
}

type DiskMetric struct {
	Drive      string `json:"drive"`
	Size       int64  `json:"size"`
	Free       int64  `json:"free"`
	Used       int64  `json:"used"`
	FormatSize string `json:"formatSize"`
	FormatFree string `json:"formatFree"`
}

// SystemInfo represents system metrics (CPU, memory, disk)
type SystemInfo struct {
	CPU    [][]float64  `json:"cpu"`    // Each entry is [timestamp, usage]
	Memory [][]float64  `json:"memory"` // Each entry is [timestamp, usage]
	Disk   []DiskMetric `json:"disk"`
}

type Metric struct {
	Timestamp    int64
	TimestampMem int64
	CPUUsage     float64
	MemoryUsage  float64
}

type AgentEscalation struct {
	AppId   uuid.UUID `json:"appId"`
	AgentId string    `json:"agentId"`
	Metric  string    `json:"metric"`
	Cause   string    `json:"cause"`
}
