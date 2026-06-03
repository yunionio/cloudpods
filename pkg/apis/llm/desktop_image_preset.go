package llm

import (
	"strings"
)

type desktopImagePresetKey struct {
	repo  string
	label string
}

type desktopImagePreset struct {
	Config  LLMImageDesktopConfig
	AppName string
}

var desktopImagePresets = map[desktopImagePresetKey]desktopImagePreset{
	{repo: "webtop", label: "ubuntu-xfce"}: {
		Config: LLMImageDesktopConfig{
			UiTitle:     "Cloudpods Desktop",
			Profile:     DesktopProfileSelkies,
			DefaultPort: DesktopDefaultPort,
			Protocol:    DesktopDefaultProtocol,
		},
		AppName: DesktopAppNameWebtopUbuntuXfce,
	},
	{repo: "webtop", label: "debian-xfce"}: {
		Config: LLMImageDesktopConfig{
			UiTitle:     "Cloudpods Desktop",
			Profile:     DesktopProfileSelkies,
			DefaultPort: DesktopDefaultPort,
			Protocol:    DesktopDefaultProtocol,
		},
		AppName: DesktopAppNameWebtopDebianXfce,
	},
	{repo: "firefox", label: "latest"}: {
		Config: LLMImageDesktopConfig{
			UiTitle:     "Cloudpods Firefox",
			Profile:     DesktopProfileSelkies,
			DefaultPort: DesktopDefaultPort,
			Protocol:    DesktopDefaultProtocol,
		},
		AppName: DesktopAppNameFirefox,
	},
	{repo: "chromium", label: "latest"}: {
		Config: LLMImageDesktopConfig{
			UiTitle:     "Cloudpods Chromium",
			Profile:     DesktopProfileSelkies,
			DefaultPort: DesktopDefaultPort,
			Protocol:    DesktopDefaultProtocol,
		},
		AppName: DesktopAppNameChromium,
	},
}

func imageRepoBase(imageName string) string {
	imageName = strings.TrimSpace(imageName)
	if i := strings.LastIndex(imageName, "/"); i >= 0 {
		return imageName[i+1:]
	}
	return imageName
}

func GetDesktopImagePreset(imageName, imageLabel string) (LLMImageDesktopConfig, bool) {
	preset, ok := getDesktopImagePreset(imageName, imageLabel)
	if !ok {
		return LLMImageDesktopConfig{}, false
	}
	return preset.Config, true
}

func getDesktopImagePreset(imageName, imageLabel string) (desktopImagePreset, bool) {
	key := desktopImagePresetKey{
		repo:  imageRepoBase(imageName),
		label: strings.TrimSpace(imageLabel),
	}
	preset, ok := desktopImagePresets[key]
	return preset, ok
}

func copyLLMImageDesktopConfig(c *LLMImageDesktopConfig) *LLMImageDesktopConfig {
	if c == nil {
		return nil
	}
	out := *c
	if len(c.ExtraEnvs) > 0 {
		out.ExtraEnvs = make(map[string]string, len(c.ExtraEnvs))
		for k, v := range c.ExtraEnvs {
			out.ExtraEnvs[k] = v
		}
	}
	return &out
}

func mergeLLMImageDesktopConfig(base, override *LLMImageDesktopConfig) *LLMImageDesktopConfig {
	if base == nil && override == nil {
		return nil
	}
	out := copyLLMImageDesktopConfig(base)
	if out == nil {
		out = &LLMImageDesktopConfig{}
	}
	if override == nil {
		return out
	}
	ov := copyLLMImageDesktopConfig(override)
	if ov.UiTitle != "" {
		out.UiTitle = ov.UiTitle
	}
	if ov.Profile != "" {
		out.Profile = ov.Profile
	}
	if ov.DefaultPort > 0 {
		out.DefaultPort = ov.DefaultPort
	}
	if ov.Protocol != "" {
		out.Protocol = ov.Protocol
	}
	if len(ov.ExtraEnvs) > 0 {
		if out.ExtraEnvs == nil {
			out.ExtraEnvs = make(map[string]string, len(ov.ExtraEnvs))
		}
		for k, v := range ov.ExtraEnvs {
			out.ExtraEnvs[k] = v
		}
	}
	return out
}

func applyDesktopConfigDefaults(cfg *LLMImageDesktopConfig) *LLMImageDesktopConfig {
	if cfg == nil {
		cfg = &LLMImageDesktopConfig{}
	}
	out := copyLLMImageDesktopConfig(cfg)
	if out.Profile == "" {
		out.Profile = DesktopProfileSelkies
	}
	if out.DefaultPort <= 0 {
		out.DefaultPort = DesktopDefaultPort
	}
	if out.Protocol == "" {
		out.Protocol = DesktopDefaultProtocol
	}
	if out.UiTitle == "" {
		out.UiTitle = "Cloudpods Desktop"
	}
	return out
}

// ResolveDesktopConfig merges YAML/API input with built-in presets for image_name:image_label.
func ResolveDesktopConfig(imageName, imageLabel string, input *LLMImageDesktopConfig) (*LLMImageDesktopConfig, error) {
	var base *LLMImageDesktopConfig
	if preset, ok := getDesktopImagePreset(imageName, imageLabel); ok {
		p := preset.Config
		base = &p
	}
	merged := mergeLLMImageDesktopConfig(base, input)
	if merged == nil {
		merged = &LLMImageDesktopConfig{}
	}
	out := applyDesktopConfigDefaults(merged)
	profile := strings.TrimSpace(out.Profile)
	if profile != "" && profile != DesktopProfileSelkies {
		return nil, ErrInvalidDesktopProfile(profile)
	}
	out.Profile = DesktopProfileSelkies
	return out, nil
}

// ResolveAppName resolves desktop app_name from explicit input or built-in presets.
func ResolveAppName(imageName, imageLabel, inputAppName string) (string, error) {
	inputAppName = NormalizeDesktopAppName(inputAppName)
	if inputAppName != "" {
		if !IsValidDesktopAppName(inputAppName) {
			return "", ErrInvalidDesktopAppName(inputAppName)
		}
		return inputAppName, nil
	}
	if preset, ok := getDesktopImagePreset(imageName, imageLabel); ok && preset.AppName != "" {
		return preset.AppName, nil
	}
	if repo := imageRepoBase(imageName); repo != "" && IsValidDesktopAppName(repo) {
		return repo, nil
	}
	return "", ErrDesktopAppNameRequired{}
}

// ErrInvalidDesktopAppName is returned when app_name is unsupported.
type ErrInvalidDesktopAppName string

func (e ErrInvalidDesktopAppName) Error() string {
	return "invalid desktop app_name " + string(e)
}

// ErrDesktopAppNameRequired is returned when app_name cannot be inferred.
type ErrDesktopAppNameRequired struct{}

func (e ErrDesktopAppNameRequired) Error() string {
	return "app_name is required for desktop images without a built-in preset"
}

// ErrInvalidDesktopProfile is returned when desktop profile is unsupported.
type ErrInvalidDesktopProfile string

func (e ErrInvalidDesktopProfile) Error() string {
	return "unsupported desktop profile " + string(e) + ", only selkies is supported"
}
