package model

import (
	"github.com/webitel/flow_manager/internal/domain/calendar"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

// Re-exports for backward compatibility.
type Timezone = calendar.Timezone

func GetMillis() int64 { return utils.GetMillis() }
