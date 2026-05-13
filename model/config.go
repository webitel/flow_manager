package model

import "github.com/webitel/flow_manager/internal/bootstrap/config"

// Re-exports for backward compatibility.
type Config = config.Config
type LogSettings = config.LogSettings
type TLSConfig = config.TLSConfig
type EslSettings = config.EslSettings
type ChatTemplatesSettings = config.ChatTemplatesSettings
type RedisSettings = config.RedisSettings
type GrpcServeSettings = config.GrpcServeSettings
type DiscoverySettings = config.DiscoverySettings
type SqlSettings = config.SqlSettings
type WebHookSettings = config.WebHookSettings
type MQSettings = config.MQSettings

const DATABASE_DRIVER_POSTGRES = config.DatabaseDriverPostgres
