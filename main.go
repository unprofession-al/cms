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

type Branch map[string]*Branch

func Walk(root string, fs billy.Filesystem) (Branch, error) {
	b := Branch{}
	elems, err := fs.ReadDir(root)
	if err != nil {
		return b, err
	}
	for _, elem := range elems {
		if elem.IsDir() {
			child, err := Walk(root+"/"+elem.Name(), fs)
			if err != nil {
				return b, err
			}
			b[elem.Name()] = &child
		} else {
			b[elem.Name()] = nil
		}
	}
	return b, nil
}

func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}
