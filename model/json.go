package model

import "github.com/webitel/flow_manager/internal/infrastructure/utils"

// Re-exports for backward compatibility.
func ToJson(src interface{}) string { return utils.ToJson(src) }

func JsonString[Bytes []byte | string](dst []byte, src Bytes, escapeHTML bool) []byte {
	return utils.JsonString(dst, src, escapeHTML)
}
