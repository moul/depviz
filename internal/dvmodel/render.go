package dvmodel

import "net/http"

func (b *Batch) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (t Tasks) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
