package dvserver

import (
	"fmt"
	"net/http"
)

func basicAuth(basicAuth string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, password, ok := r.BasicAuth()
			if !ok {
				unauthorized(w)
				return
			}

			if basicAuth == password {
				next.ServeHTTP(w, r)
				return
			}

			unauthorized(w)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, "invalid credentials\n")
}
