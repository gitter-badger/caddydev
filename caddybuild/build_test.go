package caddybuild

import (
	"testing"

	"github.com/caddyserver/buildsrv/features"
)

func TestGetPrevDirective(t *testing.T) {
	features.Registry = features.Middlewares{
		{"log", ""},
		{"gzip", ""},
		{"errors", ""},
		{"header", ""},
		{"rewrite", ""},
		{"redir", ""},
		{"ext", ""},
		{"basicauth", ""},
		{"internal", ""},
		{"proxy", ""},
		{"fastcgi", ""},
		{"websocket", ""},
		{"markdown", ""},
	}
	mids := []struct {
		name string
		prev string
	}{
		{"log", ""},
		{"gzip", "log"},
		{"errors", "gzip"},
		{"header", "errors"},
		{"rewrite", "header"},
		{"redir", "rewrite"},
		{"ext", "redir"},
		{"basicauth", "ext"},
		{"internal", "basicauth"},
		{"proxy", "internal"},
		{"fastcgi", "proxy"},
		{"websocket", "fastcgi"},
		{"markdown", "websocket"},
	}
	for i, m := range features.Registry {
		directivesPos[m.Directive] = i
	}

	for _, m := range mids {
		if prev := getPrevDirective(m.name); prev != m.prev {
			t.Errorf("Expected %v found %v", m.prev, prev)
		}
	}
}
