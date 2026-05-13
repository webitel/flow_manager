package cache_adapter

import "golang.org/x/sync/singleflight"

// singleflightGroup is a thin alias so the package has its own instance.
type singleflightGroup struct{ singleflight.Group }
