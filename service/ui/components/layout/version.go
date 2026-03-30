package layout

import (
	"fmt"
	"time"
)

// AssetVersion is a cache-busting string appended to static asset URLs.
// It changes on every server restart, forcing CDNs to fetch fresh files.
var AssetVersion = fmt.Sprintf("%x", time.Now().Unix())
