package model

import (
	"net"

	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

// Re-exports for backward compatibility.
func IsPublicIP(IP net.IP) bool          { return utils.IsPublicIP(IP) }
func GetPublicAddr() string              { return utils.GetPublicAddr() }
func GetFreePort() (int, error)          { return utils.GetFreePort() }
func GetPort() int                       { return utils.GetPort() }
func GetFreePorts(n int) ([]int, error)  { return utils.GetFreePorts(n) }
