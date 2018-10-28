package router

import yaml "gopkg.in/yaml.v2"

func (r Route) Definition() ([]byte, error) {
	return yaml.Marshal(r)
}
