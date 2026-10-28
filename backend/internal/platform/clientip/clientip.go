// Package clientip resolves the address of the client of a request, which is what the
// rate limiters count against. Behind reverse proxies the TCP peer is the proxy, so the
// real client is read from X-Forwarded-For, but only when the peer is a proxy we trust:
// any client can send that header and would otherwise pick its own address.
package clientip

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const forwardedFor = "X-Forwarded-For"

type ctxKey struct{}

// Middleware resolves the client address of every request once and exposes it through
// From. With no trusted proxies the header is never read.
func Middleware(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolve(r.RemoteAddr, r.Header.Values(forwardedFor), trusted)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, ip)))
		})
	}
}

// From returns the client address resolved by Middleware. A request that did not go
// through it gets the TCP peer, never the header.
func From(r *http.Request) string {
	if ip, ok := r.Context().Value(ctxKey{}).(string); ok {
		return ip
	}
	return peerHost(r.RemoteAddr)
}

// resolve returns the TCP peer unless it is a trusted proxy. In that case it walks the
// header from right to left, the order in which proxies append, and returns the first
// address that is not a trusted proxy. When the chain cannot be followed (a missing or
// malformed entry, or only proxies) it stays on the nearest trusted hop instead of
// guessing, so a forged header can never move the result to an address of its choosing.
func resolve(remoteAddr string, forwarded []string, trusted []netip.Prefix) string {
	peer, ok := parseAddr(peerHost(remoteAddr))
	if !ok || !contains(trusted, peer) {
		return peerHost(remoteAddr)
	}

	nearest := peer
	hops := strings.Split(strings.Join(forwarded, ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, ok := parseAddr(strings.TrimSpace(hops[i]))
		if !ok {
			break
		}
		if !contains(trusted, hop) {
			return hop.String()
		}
		nearest = hop
	}
	return nearest.String()
}

func peerHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

// parseAddr normalises IPv4-mapped IPv6 and drops the zone so one client has one key.
func parseAddr(raw string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap().WithZone(""), true
}

func contains(trusted []netip.Prefix, addr netip.Addr) bool {
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
