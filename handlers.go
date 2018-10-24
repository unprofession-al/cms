package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func (s Server) SitesHandler(res http.ResponseWriter, req *http.Request) {
	s.respond(res, req, http.StatusOK, s.sites)
}

func (s Server) StatusHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["site"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site not provided"))
		return
	}
	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s not found", name))
		return
	}

	w, err := site.repo.Worktree()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Worktree for %s could not be built: %s", name, err.Error()))
		return
	}

	status, err := w.Status()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Status for %s could not be fetched: %s", name, err.Error()))
		return
	}

	s.respond(res, req, http.StatusOK, status)
}

func (s Server) UpdateHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["site"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site not provided"))
		return
	}

	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s not found", name))
		return
	}

	w, err := site.repo.Worktree()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Worktree for %s could not be built: %s", name, err.Error()))
		return
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not pull changes of %s: %s", name, err.Error()))
		return
	}

	s.respond(res, req, http.StatusOK, "pulled")
}

func (s Server) PublishHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["site"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site not provided"))
		return
	}

	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s not found", name))
		return
	}

	// TODO: this should only run if changes are staged

	if site.Tag != "" {
		opts := &git.CreateTagOptions{
			Tagger:  s.getSignature(req),
			Message: site.Tag,
		}

		head, err := site.repo.Head()
		if err != nil {
			s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Could not get HEAD of %s: %s", name, err.Error()))
			return
		}

		err = site.repo.DeleteTag(site.Tag)
		if err != nil && err != git.ErrTagNotFound {
			s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Could not get HEAD of %s: %s", err.Error()))
			return
		}

		// TODO: Delete of remote tag must be executed

		_, err = site.repo.CreateTag(site.Tag, head.Hash(), opts)
		if err != nil {
			s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s could not be tagged: %s", name, err.Error()))
			return
		}
	}

	err := site.repo.Push(&git.PushOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not push changes of %s: %s", name, err.Error()))
		return
	}

	if site.Tag != "" {
		rs := config.RefSpec("refs/tags/*:refs/tags/*")
		err = site.repo.Push(&git.PushOptions{
			RefSpecs: []config.RefSpec{rs},
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not push tags of %s: %s", name, err.Error()))
			return
		}
	}

	s.respond(res, req, http.StatusOK, "published")
}

func (s Server) TreeHandler(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name, ok := vars["site"]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site not provided"))
		return
	}
	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s not found", name))
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
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site not provided"))
		return
	}
	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s not found", name))
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/sites/"+name+"/files")
	path = site.BaseDir + path
	file, err := site.fs.Open(path)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not read %s: %s", path, err.Error()))
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
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site not provided"))
		return
	}
	site, ok := s.sites[name]
	if !ok {
		s.respond(res, req, http.StatusNotFound, fmt.Sprintf("Site %s not found", name))
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/sites/"+name+"/files")
	path = site.BaseDir + path
	file, err := site.fs.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not read %s: %s", path, err.Error()))
		return
	}
	defer file.Close()

	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not read request body: %s", err.Error()))
		return
	}

	_, err = file.Write(b)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Could not write file %s: %s", path, err.Error()))
		return
	}

	w, err := site.repo.Worktree()
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Worktree for %s could not be built: %s", name, err.Error()))
		return
	}

	_, err = w.Add(path)
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Changes for %s could not be added: %s", name, err.Error()))
		return
	}

	_, err = w.Commit(fmt.Sprintf("Changes commited via cms for %s", path), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})
	if err != nil {
		s.respond(res, req, http.StatusInternalServerError, fmt.Sprintf("Changes for %s could not be commited: %s", name, err.Error()))
		return
	}

	s.respond(res, req, http.StatusOK, "saved")
}
