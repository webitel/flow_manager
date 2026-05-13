package model

import "github.com/webitel/flow_manager/internal/bootstrap/version"

// Re-exports for backward compatibility.
// Values are copied at init time; versions never change after startup.
var CurrentVersion = version.CurrentVersion
var BuildNumber = version.BuildNumber
