package model

import "github.com/webitel/flow_manager/internal/infrastructure/utils"

// Re-exports for backward compatibility.
func NewBool(b bool) *bool       { return utils.NewBool(b) }
func NewInt(n int) *int          { return utils.NewInt(n) }
func NewInt8(n int8) *int8       { return utils.NewInt8(n) }
func NewInt64(n int64) *int64    { return utils.NewInt64(n) }
func NewString(s string) *string { return utils.NewString(s) }
