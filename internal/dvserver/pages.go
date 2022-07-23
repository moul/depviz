package dvserver

import (
	"fmt"
	"html/template"
	"net/http"

	packr "github.com/gobuffalo/packr/v2"
)

func homepage(box *packr.Box, opts Opts) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// performance can be improved by computing the template only once, but it makes development harder
		content, err := box.FindString("index.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("500: %+v", err), http.StatusInternalServerError)
			return
		}

		tmpl, err := template.New("home").Parse(content)
		if err != nil {
			http.Error(w, fmt.Sprintf("500: %+v", err), http.StatusInternalServerError)
			return
		}

		data := opts
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, fmt.Sprintf("500: %+v", err), http.StatusInternalServerError)
		}
	}
}
