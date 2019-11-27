package dvserver

import (
	"fmt"
	"net/http"
)

func basicAuth(basicAuth string, realm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, password, ok := r.BasicAuth()
			if !ok {
				unauthorized(w, realm)
				return
			}

			if basicAuth == password {
				next.ServeHTTP(w, r)
				return
			}

			unauthorized(w, realm)
		})
	}
}

func unauthorized(w http.ResponseWriter, realm string) {
	// w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, "invalid credentials\n")
}
