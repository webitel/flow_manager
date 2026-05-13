// Package version exposes the application version string.
package version

import (
	"fmt"

	"github.com/webitel/flow_manager/model"
)

// String returns the human-readable version string "vX.Y.Z [build:N]".
func String() string {
	return fmt.Sprintf("%s [build:%s]", model.CurrentVersion, model.BuildNumber)
}
