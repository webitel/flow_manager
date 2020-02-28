package model

import "time"

var AppServiceName = "workflow"

const SchemaCacheSize = 10000
const SchemaCacheExpire = (60 * 60) * 24 // 24 hour

const AppServiceTTL = time.Second * 30
const AppDeregesterCriticalTTL = time.Second * 60
