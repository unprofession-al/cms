package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
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
	r.HandleFunc("/sites/{site}/status", s.StatusHandler).Methods("GET")
	r.HandleFunc("/sites/{site}/publish", s.PublishHandler).Methods("POST")
	r.HandleFunc("/sites/{site}/files", s.TreeHandler).Methods("GET")
	r.PathPrefix("/sites/{site}/files/").HandlerFunc(s.FileHandler).Methods("GET")
	r.PathPrefix("/sites/{site}/files/").HandlerFunc(s.FileWriteHandler).Methods("POST")

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
	if code != http.StatusOK {
		fmt.Println(data)
	}
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

func (s Server) StatusHandler(res http.ResponseWriter, req *http.Request) {
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

	w, err := site.repo.Worktree()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Worktree for %s could not be built: %s", site, err.Error()))
		return
	}

	status, err := w.Status()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Status for %s could not be fetched: %s", site, err.Error()))
		return
	}

	s.respond(res, req, http.StatusOK, status)
}

func (s Server) PublishHandler(res http.ResponseWriter, req *http.Request) {
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

	err := site.repo.Push(&git.PushOptions{})

	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not push changes of %: %s", site, err.Error()))
		return
	}

	s.respond(res, req, http.StatusOK, "published")
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

	path := strings.TrimPrefix(req.URL.Path, "/sites/"+name+"/files")
	path = site.BaseDir + path
	file, err := site.fs.Open(path)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not read %s: %s", path, err.Error()))
		return
	}
	defer file.Close()

	b := new(bytes.Buffer)
	b.ReadFrom(file)

	s.raw(res, http.StatusOK, b.Bytes())
}

func (s Server) FileWriteHandler(res http.ResponseWriter, req *http.Request) {
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

	path := strings.TrimPrefix(req.URL.Path, "/sites/"+name+"/files")
	path = site.BaseDir + path
	file, err := site.fs.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not read %s: %s", path, err.Error()))
		return
	}
	defer file.Close()

	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not read request body: %s", err.Error()))
		return
	}

	_, err = file.Write(b)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Could not write file %s: %s", path, err.Error()))
		return
	}

	w, err := site.repo.Worktree()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Worktree for %s could not be built: %s", site, err.Error()))
		return
	}
	commit, err := w.Commit(fmt.Sprintf("Changes commited via cms for %s", path), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Errorf("Changes for %s could not be commited: %s", site, err.Error()))
		return
	}

	s.respond(res, req, http.StatusOK, commit)
}
