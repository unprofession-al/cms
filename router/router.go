package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Route struct {
	H Handlers `json:"handlers" yaml:"handlers"`
	R Routes   `json:"routes" yaml:"routes"`
}

func (r Route) Populate(router *mux.Router, base string) {
	base = fmt.Sprintf("/%s/", strings.Trim(base, "/"))

	if strings.HasSuffix(base, "*/") {
		base = strings.TrimSuffix(base, "*/")
		for m, h := range r.H {
			f := h.F
			if f == nil {
				f = notImplemented
			}
			router.PathPrefix(base).HandlerFunc(f).Methods(m)
		}
		return
	}

	for m, h := range r.H {
		f := h.F
		if f == nil {
			f = notImplemented
		}
		router.Path(base).Methods(m).Handler(f)
	}

	for path, route := range r.R {
		path = fmt.Sprintf("%s%s/", base, strings.Trim(path, "/"))
		route.Populate(router, path)
	}
}

func notImplemented(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotImplemented)
	out := "Not Yet Implemented\n"
	res.Write([]byte(out))
}

type Routes map[string]Route

type Handler struct {
	Q []*QueryParam    `json:"query_params" yaml:"query_params"`
	D string           `json:"description" yaml:"description"`
	F http.HandlerFunc `json:"-" yaml:"-"`
}

type Handlers map[string]Handler

type QueryParam struct {
	N    string      `json:"name" yaml:"name"`
	D    string      `json:"default" yaml:"default"`
	Desc string      `json:"description" yaml:"description"`
	C    interface{} `json:"-" yaml:"-"`
}

func (q QueryParam) Get(r *http.Request) []string {
	values := r.URL.Query()[q.N]
	if len(values) < 1 && q.D != "" {
		values = append(values, q.D)
	}
	return values
}

func (q QueryParam) First(r *http.Request) string {
	values := r.URL.Query()[q.N]
	values = append(values, q.D)
	return values[0]
}
