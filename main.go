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
)

var (
	listener string
	static   string
	key      string
	pass     string
	repos    []string
)

func init() {
	pflag.StringVarP(&listener, "listener", "l", "0.0.0.0:8765", "ip/port to listen on")
	pflag.StringVarP(&key, "key", "k", "id_rsa", "path to ssh key")
	pflag.StringVarP(&pass, "pass", "p", "", "password of the ssh key")
	pflag.StringVarP(&static, "static", "s", "", "serve given dir as http root")
	pflag.StringSliceVarP(&repos, "repos", "r", []string{}, "repos")
}

func main() {
	pflag.Parse()

	pem, err := ioutil.ReadFile(key)
	CheckIfError(err)
	signer, err := ssh.ParsePrivateKeyWithPassphrase(pem, []byte(pass))
	CheckIfError(err)
	auth := &ssh2.PublicKeys{User: "git", Signer: signer}

	list := map[string]string{
		"unprofessional": "unprofession-al/website",
		"sontags":        "sontags/sonta.gs",
	}

	for _, param := range repos {
		tokens := strings.SplitN(param, "=", 2)
		if len(tokens) == 2 {
			list[tokens[0]] = tokens[1]
		} else {
			fmt.Fprintf(os.Stderr, "Param '%s' is invalid, must match pattern [key]=[value] ", param)
		}
	}

	repolist := map[string]billy.Filesystem{}
	for name, src := range list {
		url := fmt.Sprintf("git@github.com:%s.git", src)
		fmt.Printf("Loading %s from %s...\n", name, url)
		fs := memfs.New()
		_, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
			URL:  url,
			Auth: auth,
		})
		CheckIfError(err)
		repolist[name] = fs
	}

	s := NewServer(listener, static, repolist)
	s.Run()
}

type Node struct {
	Name     string `json:"name"`
	IsDir    bool   `json:"is_dir"`
	FullPath string `json:"full_path"`
	Children []Node `json:"children"`
}

func WalkNode(path string, fs billy.Filesystem) (Node, error) {
	n := Node{
		Children: []Node{},
		FullPath: path,
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
			c, err := WalkNode(elemName, fs)
			if err != nil {
				return n, err
			}
			n.Children = append(n.Children, c)
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
