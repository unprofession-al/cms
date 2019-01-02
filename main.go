package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	ssh2 "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/yaml.v2"
)

var app App

type App struct {
	listener string
	static   string
	key      string
	pass     string
	config   string
}

func init() {
	pflag.StringVarP(&app.listener, "listener", "l", "0.0.0.0:8765", "ip/port to listen on")
	pflag.StringVarP(&app.key, "key", "k", "id_rsa", "path to ssh key")
	pflag.StringVarP(&app.pass, "pass", "p", "", "password of the ssh key")
	pflag.StringVarP(&app.static, "static", "s", "", "serve given dir as http root")
	pflag.StringVarP(&app.config, "config", "c", "cms.yaml", "configuration file")
}

type Site struct {
	Git        string   `yaml:"git"`
	Key        string   `yaml:"key"`
	BaseDir    string   `yaml:"baseDir"`
	ExtAllowed []string `yaml:"extensionsAllowed"`
	fs         billy.Filesystem
	repo       *git.Repository
}

type Config struct {
	Sites map[string]*Site `yaml:"sites"`
}

func NewConfig(path string) (Config, error) {
	c := Config{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("Error while reading configuration file '%s': %s", path, err.Error())
	}

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return c, fmt.Errorf("Error while unmarshalling configuration '%s' from yaml: %s", path, err.Error())
	}

	for _, site := range c.Sites {
		site.BaseDir = strings.Trim(site.BaseDir, "/")
	}
	return c, nil
}

func main() {
	pflag.Parse()

	c, err := NewConfig(app.config)
	CheckIfError(err)

	pem, err := ioutil.ReadFile(app.key)
	CheckIfError(err)
	signer, err := ssh.ParsePrivateKeyWithPassphrase(pem, []byte(app.pass))
	CheckIfError(err)
	auth := &ssh2.PublicKeys{User: "git", Signer: signer}

	for name, site := range c.Sites {
		fmt.Printf("Loading %s from %s...\n", name, site.Git)
		fs := memfs.New()
		c.Sites[name].repo, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
			URL:      site.Git,
			Auth:     auth,
			Progress: os.Stdout,
		})
		CheckIfError(err)
		c.Sites[name].fs = fs
	}

	s := NewServer(app.listener, app.static, c.Sites)
	s.run()
}

type Node struct {
	Name     string `json:"name"`
	IsDir    bool   `json:"is_dir"`
	FullPath string `json:"full_path"`
	Children []Node `json:"children"`
}

func WalkNode(path string, fs billy.Filesystem, trim string, extAllowed []string) (Node, error) {
	n := Node{
		Children: []Node{},
		FullPath: strings.TrimPrefix(path, trim),
	}

	e, err := fs.Stat(path)
	if err != nil {
		return n, err
	}

	n.Name = e.Name()
	n.IsDir = e.IsDir()

	if n.IsDir {
		elems, err := fs.ReadDir(path)
		if err != nil {
			return n, err
		}
		for _, elem := range elems {
			elemName := fmt.Sprintf("%s/%s", path, elem.Name())
			c, err := WalkNode(elemName, fs, trim, extAllowed)
			if err != nil {
				return n, err
			}

			// if name is "" either the folder is empty or the extention is not allowed
			if c.Name != "" {
				n.Children = append(n.Children, c)
			}
		}

		// don't return if folder is empty
		if len(n.Children) == 0 {
			return Node{}, nil
		}
	} else {
		allowed := false
		for _, ext := range extAllowed {
			if strings.HasSuffix(n.Name, ext) {
				allowed = true
			}
		}
		if !allowed {
			return Node{}, nil
		}
	}
	return n, nil
}

func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}
