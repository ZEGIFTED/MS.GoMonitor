package internal

import (
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
	"github.com/google/uuid"
)

type ServiceAlertEvent struct {
	ServiceName     string
	SystemMonitorId uuid.UUID `json:"systemMonitorId"`
	Message         string
	Device          string
	Severity        string
	Timestamp       time.Time
	AgentRepository repository.AgentRepository
	ServiceStats    mstypes.ProcessResponse
	AgentAPI        string
}
