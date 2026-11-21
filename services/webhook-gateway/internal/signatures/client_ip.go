package signatures

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func GetClientIP(r *http.Request, trustedProxy bool) (string, error) {
	peerIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("parse remote address: %w", err)
	}

	ip := peerIP

	if trustedProxy {
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			// Only safe if Traefik has validated the forwarded chain.
			ip = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", fmt.Errorf("parse client IP: %w", err)
	}

	return addr.Unmap().String(), nil
}
