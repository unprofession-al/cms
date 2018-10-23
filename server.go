package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"gopkg.in/yaml.v2"
)

type Server struct {
	sites    map[string]*Site
	listener string
	handler  http.Handler
}

func NewServer(listener, static string, sites map[string]*Site) Server {
	s := Server{
		listener: listener,
		sites:    sites,
	}

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/sites/", s.SitesHandler).Methods("GET")
	r.HandleFunc("/sites/{site}", s.TreeHandler).Methods("GET")
	r.PathPrefix("/sites/{site}/").HandlerFunc(s.FileHandler).Methods("GET")

	if static != "" {
		r.PathPrefix("/").Handler(http.FileServer(http.Dir(static)))
	}

	s.handler = alice.New().Then(r)
	return s
}

func (s Server) Run() {
	fmt.Printf("Serving cms at http://%s\nPress CTRL-c to stop...\n", s.listener)
	log.Fatal(http.ListenAndServe(s.listener, s.handler))
}

func (s Server) respond(res http.ResponseWriter, req *http.Request, code int, data interface{}) {
	var err error
	var errMesg []byte
	var out []byte

	f := "json"
	format := req.URL.Query()["f"]
	if len(format) > 0 {
		f = format[0]
	}

	if f == "yaml" {
		res.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		out, err = yaml.Marshal(data)
		errMesg = []byte("--- error: failed while rendering data to yaml")
	} else {
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		out, err = json.Marshal(data)
		errMesg = []byte("{ 'error': 'failed while rendering data to json' }")
	}

	if err != nil {
		out = errMesg
		code = http.StatusInternalServerError
	}
	res.WriteHeader(code)
	res.Write(out)
}

func (s Server) raw(res http.ResponseWriter, code int, data []byte) {
	res.WriteHeader(code)
	res.Write(data)
}

func (s Server) SitesHandler(res http.ResponseWriter, req *http.Request) {
	s.respond(res, req, http.StatusOK, s.sites)
}
func (s Server) TreeHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["site"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Site not provided"))
		return
	}
	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Site %s not found", site))
		return
	}

	tree, err := WalkNode(site.BaseDir, site.fs, site.BaseDir)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, err.Error())
		return
	}

	s.respond(res, req, http.StatusOK, tree)
}

func (s Server) FileHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	name, ok := vars["site"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Site not provided"))
		return
	}
	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Site %s not found", name))
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/sites/"+name)
	path = site.BaseDir + path
	file, err := site.fs.Open(path)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not read %s: %s", path, err.Error()))
		return
	}

	b := new(bytes.Buffer)
	b.ReadFrom(file)

	s.raw(res, http.StatusOK, b.Bytes())
}
