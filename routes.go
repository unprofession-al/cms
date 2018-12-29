package main

import (
	r "github.com/unprofession-al/routing"
)

func (s Server) sitesRoutes() r.Route {
	return r.Route{
		H: r.Handlers{"GET": r.Handler{F: s.SitesHandler, Q: []*r.QueryParam{formatParam}}},
		R: r.Routes{
			"{site}": {
				R: r.Routes{
					"status":  {H: r.Handlers{"GET": r.Handler{F: s.StatusHandler, Q: []*r.QueryParam{formatParam}}}},
					"publish": {H: r.Handlers{"PUT": r.Handler{F: s.PublishHandler, Q: []*r.QueryParam{formatParam}}}},
					"update":  {H: r.Handlers{"PUT": r.Handler{F: s.UpdateHandler, Q: []*r.QueryParam{formatParam}}}},
					"files": {
						H: r.Handlers{"GET": r.Handler{F: s.TreeHandler, Q: []*r.QueryParam{formatParam}}},
						R: r.Routes{
							"*": {
								H: r.Handlers{
									"GET":  {F: s.FileHandler, Q: []*r.QueryParam{mdParam}},
									"POST": {F: s.FileWriteHandler, Q: []*r.QueryParam{mdParam}},
								},
							},
						},
					},
				},
			},
		},
	}
}

var formatParam = &r.QueryParam{
	N:    "f",
	D:    "json",
	Desc: "format of the output, can be 'yaml' or 'json'",
}

var mdParam = &r.QueryParam{
	N: "o",
	D: "all",
	Desc: `define if only one part of the markdown file is requested,
can be 'fm' for frontmatter, 'md' for markdown, all for everything`,
}
