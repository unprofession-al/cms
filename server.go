package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
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
	r.HandleFunc("/sites/{site}/publish", s.PublishHandler).Methods("PUT")
	r.HandleFunc("/sites/{site}/update", s.UpdateHandler).Methods("PUT")
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

func (s Server) getSignature(req *http.Request) *object.Signature {
	return &object.Signature{
		Name:  "John Doe",
		Email: "john@doe.org",
		When:  time.Now(),
	}
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

func splitMarkdown(data []byte) (frontmatter, markdown []byte, err error) {
	d := []byte("---\n")
	if bytes.Count(data, d) < 2 {
		errors.New("Input does not look like markdown with frontmatter")
		return
	}
	parts := bytes.SplitN(data, d, 3)
	frontmatter = parts[1]
	markdown = parts[2]
	return
}

func joinMarkdown(file, data []byte, section string) (out []byte, err error) {
	d := []byte("---\n")
	if bytes.Count(file, d) < 2 {
		errors.New("Input does not look like markdown with frontmatter")
		return
	}
	parts := bytes.SplitN(file, d, 3)

	frontmatter := parts[1]
	markdown := parts[2]

	if section == "fm" {
		frontmatter = data
	} else if section == "md" {
		markdown = data
	} else {
		fmt.Errorf("Sectoin %s is unknown", section)
		return
	}

	concat := [][]byte{d, frontmatter, d, markdown}
	out = bytes.Join(concat, []byte(""))

	return
}
