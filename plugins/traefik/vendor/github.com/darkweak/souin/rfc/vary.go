package rfc

import (
	"io"
	"io/ioutil"
	"net/http"
)

// varyMatches will return false unless all of the cached values for the headers listed in Vary
// match the new request
func varyMatches(cachedResp *http.Response, req *http.Request) bool {
	for _, header := range headerAllCommaSepValues(cachedResp.Header) {
		header = http.CanonicalHeaderKey(header)
		if header == "" || req.Header.Get(header) == "" {
			return false
		}
	}
	return true
}

func validateVary(req *http.Request, resp *http.Response, key string, t *VaryTransport) bool {
	if resp != nil {
		variedHeaders := headerAllCommaSepValues(resp.Header)
		cacheKey := key
		if len(variedHeaders) > 0 {
			cacheKey = GetVariedCacheKey(req, variedHeaders)
		}
		switch req.Method {
		case http.MethodGet:
			// SetCache before EOF to set cache with a partial response then override the cache with the full one once it reach EOF
			t.SetCache(cacheKey, resp)
			_ = t.SurrogateStorage.Store(req, cacheKey)
			resp.Header.Set("Cache-Status", "Souin; fwd=uri-miss: stored")
			// Delay caching until EOF is reached.
			resp.Body = &cachingReadCloser{
				R: resp.Body,
				OnEOF: func(r io.Reader) {
					re := *resp
					re.Body = ioutil.NopCloser(r)
					t.SetCache(cacheKey, &re)
					_ = t.SurrogateStorage.Store(req, cacheKey)
					go func() {
						t.CoalescingLayerStorage.Delete(cacheKey)
					}()
				},
			}
		}
		return true
	}

	return false
}
