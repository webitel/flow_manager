package model

import "github.com/webitel/flow_manager/internal/infrastructure/utils"

// Re-exports for backward compatibility.
func NewId() string                                                          { return utils.NewId() }
func InterfaceToString(v interface{}) string                                { return utils.InterfaceToString(v) }
func StringValueFromMap(n string, p map[string]interface{}, d string) string { return utils.StringValueFromMap(n, p, d) }
func IntValueFromMap(n string, p map[string]interface{}, d int) int         { return utils.IntValueFromMap(n, p, d) }
func InterfaceToJson(i interface{}) []byte                                  { return utils.InterfaceToJson(i) }
func UrlEncoded(str string) string                                          { return utils.UrlEncoded(str) }
