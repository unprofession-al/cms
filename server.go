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
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/yaml.v2"
)

type Server struct {
	repos    map[string]billy.Filesystem
	listener string
	handler  http.Handler
}

func NewServer(listener, static string, repos map[string]billy.Filesystem) Server {
	s := Server{
		listener: listener,
		repos:    repos,
	}

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/{repo}", s.TreeHandler).Methods("GET")
	r.PathPrefix("/{repo}/").HandlerFunc(s.FileHandler).Methods("GET")

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

func (s Server) TreeHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	repo, ok := vars["repo"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Repo param not provided"))
		return
	}
	fs, ok := s.repos[repo]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Repo %s not found", repo))
		return
	}

	tree, err := WalkNode(".", fs)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, err.Error())
		return
	}

	s.respond(res, req, http.StatusOK, tree)
}

func (s Server) FileHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	repo, ok := vars["repo"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Repo param not provided"))
		return
	}
	fs, ok := s.repos[repo]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Errorf("Repo %s not found", repo))
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/"+repo)
	file, err := fs.Open(path)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not read %s: %s", file, err.Error()))
		return
	}

	b := new(bytes.Buffer)
	b.ReadFrom(file)

	s.raw(res, http.StatusOK, b.Bytes())
}
