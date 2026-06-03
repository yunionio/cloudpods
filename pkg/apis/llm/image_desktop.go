package llm

import (
	"reflect"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

const (
	DesktopProfileSelkies  = "selkies"
	DesktopDefaultPort     = 3001
	DesktopDefaultProtocol = "https"

	DesktopAppNameWebtopUbuntuXfce = "webtop-ubuntu-xfce"
	DesktopAppNameWebtopDebianXfce = "webtop-debian-xfce"
	DesktopAppNameFirefox          = "firefox"
	DesktopAppNameChromium         = "chromium"
)

var validDesktopAppNames = map[string]struct{}{
	DesktopAppNameWebtopUbuntuXfce: {},
	DesktopAppNameWebtopDebianXfce: {},
	DesktopAppNameFirefox:          {},
	DesktopAppNameChromium:         {},
}

// IsValidDesktopAppName reports whether name is a supported desktop application identifier.
// Built-in names (firefox, chromium, webtop-*, etc.) and LinuxServer-style custom ids (e.g. steam) are allowed.
func IsValidDesktopAppName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return false
	}
	if _, ok := validDesktopAppNames[name]; ok {
		return true
	}
	return isLinuxServerStyleAppName(name)
}

func isLinuxServerStyleAppName(name string) bool {
	for i, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			if i == 0 {
				return false
			}
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			if i == 0 || i == len(name)-1 {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// NormalizeDesktopAppName lowercases ASCII letters for storage and comparison.
func NormalizeDesktopAppName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r - 'A' + 'a')
		} else if r <= unicode.MaxASCII {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// LLMImageDesktopConfig holds LinuxServer Selkies desktop metadata for llm_type=desktop images.
type LLMImageDesktopConfig struct {
	UiTitle     string            `json:"ui_title,omitempty"`
	Profile     string            `json:"profile,omitempty"`
	DefaultPort int               `json:"default_port,omitempty"`
	Protocol    string            `json:"protocol,omitempty"`
	ExtraEnvs   map[string]string `json:"extra_envs,omitempty"`
}

func (c *LLMImageDesktopConfig) String() string {
	return jsonutils.Marshal(c).String()
}

func (c *LLMImageDesktopConfig) IsZero() bool {
	if c == nil {
		return true
	}
	return c.UiTitle == "" && c.Profile == "" &&
		c.DefaultPort == 0 && c.Protocol == "" && len(c.ExtraEnvs) == 0
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(new(LLMImageDesktopConfig)), func() gotypes.ISerializable {
		return new(LLMImageDesktopConfig)
	})
}
