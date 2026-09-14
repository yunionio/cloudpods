// Package ref is used to define references
// References default to remote registry references (registry:port/repo:tag)
// Schemes can be included in front of the reference for different reference types
package ref

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

const (
	dockerLibrary = "library"
	// DockerRegistry is the name resolved in docker images on Hub
	dockerRegistry = "docker.io"
	// DockerRegistryLegacy is the name resolved in docker images on Hub
	dockerRegistryLegacy = "index.docker.io"
	// DockerRegistryDNS is the host to connect to for Hub
	dockerRegistryDNS = "registry-1.docker.io"
)

var (
	hostPartS = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)`
	// host with port allows a short name in addition to hostDomainS
	hostPortS = `(?:` + hostPartS + `(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)*` + regexp.QuoteMeta(`.`) + `?` + regexp.QuoteMeta(`:`) + `[0-9]+)`
	// hostname may be ip, fqdn (example.com), or trailing dot (example.)
	hostDomainS = `(?:` + hostPartS + `(?:(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)+` + regexp.QuoteMeta(`.`) + `?|` + regexp.QuoteMeta(`.`) + `))`
	hostUpperS  = `(?:[a-zA-Z0-9]*[A-Z][a-zA-Z0-9-]*[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[A-Z][a-zA-Z0-9]*)`
	registryS   = `(?:` + hostDomainS + `|` + hostPortS + `|` + hostUpperS + `|localhost(?:` + regexp.QuoteMeta(`:`) + `[0-9]+))`
	repoPartS   = `[a-z0-9]+(?:(?:[_.]|__|[-]*)[a-z0-9]+)*`
	pathS       = `[/a-zA-Z0-9_\-. ]+`
	tagS        = `[\w][\w.-]{0,127}`
	digestS     = `[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}`
	refRE       = regexp.MustCompile(`^(?:(` + registryS + `)` + regexp.QuoteMeta(`/`) + `)?` +
		`(` + repoPartS + `(?:` + regexp.QuoteMeta(`/`) + repoPartS + `)*)` +
		`(?:` + regexp.QuoteMeta(`:`) + `(` + tagS + `))?` +
		`(?:` + regexp.QuoteMeta(`@`) + `(` + digestS + `))?$`)
	schemeRE = regexp.MustCompile(`^([a-z]+)://(.+)$`)
	pathRE   = regexp.MustCompile(`^(` + pathS + `)` +
		`(?:` + regexp.QuoteMeta(`:`) + `(` + tagS + `))?` +
		`(?:` + regexp.QuoteMeta(`@`) + `(` + digestS + `))?$`)
)

// Ref reference to a registry/repository
// If the tag or digest is available, it's also included in the reference.
// Reference itself is the unparsed string.
// While this is currently a struct, that may change in the future and access
// to contents should not be assumed/used.
type Ref struct {
	Scheme     string
	Reference  string // unparsed string
	Registry   string // server, host:port
	Repository string // path on server
	Tag        string
	Digest     string
	Path       string
}

// New returns a reference based on the scheme, defaulting to a
func New(parse string) (Ref, error) {
	scheme := ""
	path := parse
	matchScheme := schemeRE.FindStringSubmatch(parse)
	if len(matchScheme) == 3 {
		scheme = matchScheme[1]
		path = matchScheme[2]
	}
	ret := Ref{
		Scheme:    scheme,
		Reference: parse,
	}
	switch scheme {
	case "":
		ret.Scheme = "reg"
		matchRef := refRE.FindStringSubmatch(path)
		if matchRef == nil || len(matchRef) < 5 {
			if refRE.FindStringSubmatch(strings.ToLower(path)) != nil {
				return Ref{}, fmt.Errorf("invalid reference \"%s\", repo must be lowercase", path)
			}
			return Ref{}, fmt.Errorf("invalid reference \"%s\"", path)
		}
		ret.Registry = matchRef[1]
		ret.Repository = matchRef[2]
		ret.Tag = matchRef[3]
		ret.Digest = matchRef[4]

		// handle localhost use case since it matches the regex for a repo path entry
		repoPath := strings.Split(ret.Repository, "/")
		if ret.Registry == "" && repoPath[0] == "localhost" {
			ret.Registry = repoPath[0]
			ret.Repository = strings.Join(repoPath[1:], "/")
		}
		switch ret.Registry {
		case "", dockerRegistryDNS, dockerRegistryLegacy:
			ret.Registry = dockerRegistry
		}
		if ret.Registry == dockerRegistry && !strings.Contains(ret.Repository, "/") {
			ret.Repository = dockerLibrary + "/" + ret.Repository
		}
		if ret.Tag == "" && ret.Digest == "" {
			ret.Tag = "latest"
		}
		if ret.Repository == "" {
			return Ref{}, fmt.Errorf("invalid reference \"%s\"", path)
		}

	case "ocidir", "ocifile":
		matchPath := pathRE.FindStringSubmatch(path)
		if matchPath == nil || len(matchPath) < 2 || matchPath[1] == "" {
			return Ref{}, fmt.Errorf("invalid path for scheme \"%s\": %s", scheme, path)
		}
		ret.Path = matchPath[1]
		if len(matchPath) > 2 && matchPath[2] != "" {
			ret.Tag = matchPath[2]
		}
		if len(matchPath) > 3 && matchPath[3] != "" {
			ret.Digest = matchPath[3]
		}

	default:
		return Ref{}, fmt.Errorf("unhandled reference scheme \"%s\" in \"%s\"", scheme, parse)
	}
	return ret, nil
}

// CommonName outputs a parsable name from a reference
func (r Ref) CommonName() string {
	cn := ""
	switch r.Scheme {
	case "reg":
		if r.Registry != "" {
			cn = r.Registry + "/"
		}
		if r.Repository == "" {
			return ""
		}
		cn = cn + r.Repository
		if r.Tag != "" {
			cn = cn + ":" + r.Tag
		}
		if r.Digest != "" {
			cn = cn + "@" + r.Digest
		}
	case "ocidir":
		cn = fmt.Sprintf("ocidir://%s", r.Path)
		if r.Tag != "" {
			cn = cn + ":" + r.Tag
		}
		if r.Digest != "" {
			cn = cn + "@" + r.Digest
		}
	}
	return cn
}

// IsZero returns true if ref is unset
func (r Ref) IsZero() bool {
	if r.Scheme == "" && r.Registry == "" && r.Repository == "" && r.Path == "" && r.Tag == "" && r.Digest == "" {
		return true
	}
	return false
}

// ToReg converts a reference to a registry like syntax
func (r Ref) ToReg() Ref {
	switch r.Scheme {
	case "ocidir":
		r.Scheme = "reg"
		r.Registry = "localhost"
		// clean the path to strip leading ".."
		r.Repository = path.Clean("/" + r.Path)[1:]
		r.Repository = strings.ToLower(r.Repository)
		// convert any unsupported characters to "-" in the path
		re := regexp.MustCompile(`[^/a-z0-9]+`)
		r.Repository = string(re.ReplaceAll([]byte(r.Repository), []byte("-")))
	}
	return r
}

// EqualRegistry compares the registry between two references
func EqualRegistry(a, b Ref) bool {
	if a.Scheme != b.Scheme {
		return false
	}
	switch a.Scheme {
	case "reg":
		return a.Registry == b.Registry
	case "ocidir":
		return a.Path == b.Path
	case "":
		// both undefined
		return true
	default:
		return false
	}
}

// EqualRepository compares the repository between two references
func EqualRepository(a, b Ref) bool {
	if a.Scheme != b.Scheme {
		return false
	}
	switch a.Scheme {
	case "reg":
		return a.Registry == b.Registry && a.Repository == b.Repository
	case "ocidir":
		return a.Path == b.Path
	case "":
		// both undefined
		return true
	default:
		return false
	}
}
