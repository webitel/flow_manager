package model

import "time"

var AppServiceName = "11workflow"

const HeaderFromServiceName = "From-Service"

const SchemaCacheSize = 10000
const SchemaCacheExpire = (60 * 60) * 24 // 24 hour

const AppServiceTTL = time.Second * 30
const AppDeregisterCriticalTTL = time.Second * 60
