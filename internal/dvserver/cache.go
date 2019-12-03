package dvserver

import (
	"bytes"
	"encoding/gob"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"

	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// this cache middleware was inspired by https://github.com/victorspringer/http-cache

func cacheMiddleware(next http.Handler, c *cache.Cache, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			err := func() error {
				sortURLParams(r.URL)
				h := fnv.New64a()
				key := string(h.Sum([]byte(r.URL.String())))
				res, found := c.Get(key)
				if found {
					var response cached
					dec := gob.NewDecoder(bytes.NewReader(res.([]byte)))
					err := dec.Decode(&response)
					if err != nil {
						return err
					}
					for k, v := range response.Header {
						w.Header().Set(k, strings.Join(v, ","))
					}
					_, err = w.Write(response.Value)
					return err
				}

				rec := httptest.NewRecorder()
				next.ServeHTTP(rec, r)
				result := rec.Result()

				statusCode := result.StatusCode
				value := rec.Body.Bytes()
				if statusCode < 400 {
					response := cached{
						Value:  value,
						Header: result.Header,
					}
					var b bytes.Buffer
					enc := gob.NewEncoder(&b)
					err := enc.Encode(&response)
					if err != nil {
						return err
					}
					c.Set(key, b.Bytes(), cache.DefaultExpiration)
				}
				for k, v := range result.Header {
					w.Header().Set(k, strings.Join(v, ","))
				}
				w.WriteHeader(statusCode)
				_, err := w.Write(value)
				return err
			}()
			if err != nil {
				logger.Warn("caching error", zap.Error(err))
			} else {
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type cached struct {
	Value  []byte
	Header http.Header
}

func sortURLParams(URL *url.URL) {
	params := URL.Query()
	for _, param := range params {
		sort.Slice(param, func(i, j int) bool {
			return param[i] < param[j]
		})
	}
	URL.RawQuery = params.Encode()
}
