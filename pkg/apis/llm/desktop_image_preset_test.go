package llm

import (
	"testing"
)

func TestResolveDesktopConfigPreset(t *testing.T) {
	cfg, err := ResolveDesktopConfig(
		"registry.cn-beijing.aliyuncs.com/cloudpods/webtop",
		"ubuntu-xfce",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UiTitle != "Cloudpods Desktop" {
		t.Fatalf("ui_title = %q", cfg.UiTitle)
	}
	if cfg.DefaultPort != DesktopDefaultPort {
		t.Fatalf("default_port = %d", cfg.DefaultPort)
	}
	if cfg.Profile != DesktopProfileSelkies {
		t.Fatalf("profile = %q", cfg.Profile)
	}
}

func TestResolveDesktopConfigMergeInput(t *testing.T) {
	cfg, err := ResolveDesktopConfig(
		"registry.cn-beijing.aliyuncs.com/cloudpods/webtop",
		"ubuntu-xfce",
		&LLMImageDesktopConfig{UiTitle: "Custom Title"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UiTitle != "Custom Title" {
		t.Fatalf("ui_title = %q", cfg.UiTitle)
	}
}

func TestResolveDesktopConfigInvalidProfile(t *testing.T) {
	_, err := ResolveDesktopConfig("webtop", "ubuntu-xfce", &LLMImageDesktopConfig{Profile: "vnc"})
	if err == nil {
		t.Fatal("expected error for invalid profile")
	}
}

func TestGetDesktopImagePresetFirefox(t *testing.T) {
	cfg, ok := GetDesktopImagePreset("lscr.io/linuxserver/firefox", "latest")
	if !ok {
		t.Fatal("expected preset")
	}
	if cfg.UiTitle != "Cloudpods Firefox" {
		t.Fatalf("ui_title = %q", cfg.UiTitle)
	}
}

func TestResolveAppNamePreset(t *testing.T) {
	name, err := ResolveAppName("lscr.io/linuxserver/firefox", "latest", "")
	if err != nil {
		t.Fatal(err)
	}
	if name != DesktopAppNameFirefox {
		t.Fatalf("app_name = %q", name)
	}
}

func TestResolveAppNameExplicit(t *testing.T) {
	name, err := ResolveAppName("custom/image", "latest", DesktopAppNameChromium)
	if err != nil {
		t.Fatal(err)
	}
	if name != DesktopAppNameChromium {
		t.Fatalf("app_name = %q", name)
	}
}

func TestResolveAppNameSteam(t *testing.T) {
	name, err := ResolveAppName("lscr.io/linuxserver/steam", "latest", "steam")
	if err != nil {
		t.Fatal(err)
	}
	if name != "steam" {
		t.Fatalf("app_name = %q", name)
	}
}

func TestResolveAppNameInferFromImageRepo(t *testing.T) {
	name, err := ResolveAppName("lscr.io/linuxserver/steam", "latest", "")
	if err != nil {
		t.Fatal(err)
	}
	if name != "steam" {
		t.Fatalf("app_name = %q", name)
	}
}

func TestResolveAppNameInvalid(t *testing.T) {
	for _, invalid := range []string{"", "Bad Name", "-leading", "trailing-", "UPPER"} {
		if IsValidDesktopAppName(invalid) {
			t.Fatalf("expected invalid app_name %q", invalid)
		}
	}
	_, err := ResolveAppName("firefox", "latest", "Bad Name")
	if err == nil {
		t.Fatal("expected error for invalid app_name")
	}
}

func TestResolveAppNameRequired(t *testing.T) {
	_, err := ResolveAppName("", "latest", "")
	if err == nil {
		t.Fatal("expected error when app_name cannot be inferred")
	}
}

func TestIsValidDesktopAppNameBuiltin(t *testing.T) {
	if !IsValidDesktopAppName(DesktopAppNameFirefox) {
		t.Fatal("firefox should be valid")
	}
	if !IsValidDesktopAppName("unknown-app") {
		t.Fatal("linuxserver-style id should be valid")
	}
}
