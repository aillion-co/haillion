package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
)

type GatewayRouter struct {
	routes   map[string]*httputil.ReverseProxy
	prefixes []string
}

func NewGatewayRouter(serviceURLs map[string]string) (*GatewayRouter, error) {
	routes := make(map[string]*httputil.ReverseProxy)

	for prefix, rawURL := range serviceURLs {
		if rawURL == "" {
			continue
		}
		targetURL, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.Director = nil //nolint:staticcheck // SA1019: NewSingleHostReverseProxy sets Director, so we must nil it to use Rewrite without conflict
		prefixCopy := prefix // Capture loop variable

		proxy.Rewrite = func(pr *httputil.ProxyRequest) {
			pr.SetURL(targetURL)
			// Strip the gateway path prefix when proxying to downstream services
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, prefixCopy)
			if !strings.HasPrefix(pr.Out.URL.Path, "/") {
				pr.Out.URL.Path = "/" + pr.Out.URL.Path
			}
		}

		routes[prefix] = proxy
	}

	prefixes := make([]string, 0, len(routes))
	for prefix := range routes {
		prefixes = append(prefixes, prefix)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})

	return &GatewayRouter{routes: routes, prefixes: prefixes}, nil
}

func (g *GatewayRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for frontend integration
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, prefix := range g.prefixes {
		if strings.HasPrefix(r.URL.Path, prefix) {
			proxy := g.routes[prefix]
			proxy.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, "gateway: route not found", http.StatusNotFound)
}
