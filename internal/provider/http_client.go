package provider

import (
	"net/http"
	"time"
)

var providerHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
}
