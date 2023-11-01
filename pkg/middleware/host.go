package middleware

import (
	"net/http"
	"strings"
)

func HostInAllowed(r *http.Request, hosts string) bool {
	allowed := false
	for _, host := range strings.Split(hosts, ",") {
		if host == r.Host {
			allowed = true
			break
		}
	}
	return allowed
}
