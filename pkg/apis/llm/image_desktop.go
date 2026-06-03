package llm

import (
	"reflect"

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
func IsValidDesktopAppName(name string) bool {
	_, ok := validDesktopAppNames[name]
	return ok
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
