package guestman

import (
	"context"
	"encoding/json"
	"fmt"
	goruntime "runtime"
	"strconv"
	"strings"

	"github.com/vishvananda/netns"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func (cr *containerRunner) RunInPodNetNS(podId string, run func() error) error {
	srv, ok := cr.manager.GetServer(podId)
	if !ok {
		return errors.Errorf("server %s not found", podId)
	}
	pod, ok := srv.(*sPodGuestInstance)
	if !ok {
		return errors.Errorf("server %s is not a pod instance", podId)
	}
	netNSPath, err := pod.getPodNetNSPath(context.Background())
	if err != nil {
		return errors.Wrap(err, "get pod netns path")
	}
	log.Infof("[startup-probe-trace] pod netns path pod=%s path=%s", podId, netNSPath)
	return runInNetNSPath(netNSPath, run)
}

func (s *sPodGuestInstance) getPodNetNSPath(ctx context.Context) (string, error) {
	criId := s.GetCRIId()
	if criId == "" {
		return "", errors.Errorf("pod %s missing cri id", s.GetId())
	}
	status, err := s.getCRI().GetRuntimeClient().PodSandboxStatus(ctx, &runtimeapi.PodSandboxStatusRequest{
		PodSandboxId: criId,
		Verbose:      true,
	})
	if err != nil {
		return "", errors.Wrapf(err, "PodSandboxStatus %s", criId)
	}
	return sandboxNetNSPathFromStatus(status)
}

func runInNetNSPath(netNSPath string, run func() error) (retErr error) {
	netNSPath = strings.TrimSpace(netNSPath)
	if netNSPath == "" {
		return errors.Errorf("netns path is empty")
	}

	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()

	origin, err := netns.Get()
	if err != nil {
		return errors.Wrap(err, "get current netns")
	}
	defer origin.Close()

	target, err := netns.GetFromPath(netNSPath)
	if err != nil {
		return errors.Wrapf(err, "get target netns %s", netNSPath)
	}
	defer target.Close()

	if !origin.Equal(target) {
		if err := netns.Set(target); err != nil {
			return errors.Wrapf(err, "set netns %s", netNSPath)
		}
		defer func() {
			if err := netns.Set(origin); err != nil && retErr == nil {
				retErr = errors.Wrap(err, "restore original netns")
			}
		}()
	}

	return run()
}

func sandboxNetNSPathFromStatus(status *runtimeapi.PodSandboxStatusResponse) (string, error) {
	if status == nil {
		return "", errors.Errorf("pod sandbox status is nil")
	}
	finder := &sandboxNetNSFinder{}
	for key, value := range status.GetInfo() {
		if decoded, ok := decodeSandboxInfoJSON(value); ok {
			finder.inspectTopLevel(key, decoded)
		} else {
			finder.inspectTopLevel(key, value)
		}
	}
	if finder.pid > 0 {
		return fmt.Sprintf("/proc/%d/ns/net", finder.pid), nil
	}
	if finder.path != "" {
		return finder.path, nil
	}
	return "", errors.Errorf("pod sandbox status does not include netns path or sandbox pid")
}

type sandboxNetNSFinder struct {
	path string
	pid  int
}

func (f *sandboxNetNSFinder) inspectTopLevel(key string, value interface{}) {
	f.inspectScalar(key, value, true)
	switch typedValue := value.(type) {
	case map[string]interface{}:
		if isNetworkNamespaceEntry(typedValue) {
			if pathValue, ok := namespaceEntryPath(typedValue); ok {
				f.path = pathValue
				return
			}
		}
		for nestedKey, nestedValue := range typedValue {
			f.inspect(nestedKey, nestedValue, true)
		}
	case []interface{}:
		for _, nestedValue := range typedValue {
			f.inspect("", nestedValue, false)
		}
	}
}

func (f *sandboxNetNSFinder) inspect(key string, value interface{}, allowPID bool) {
	if f.path != "" && f.pid > 0 {
		return
	}

	f.inspectScalar(key, value, allowPID)

	switch typedValue := value.(type) {
	case map[string]interface{}:
		if isNetworkNamespaceEntry(typedValue) {
			if pathValue, ok := namespaceEntryPath(typedValue); ok {
				f.path = pathValue
				return
			}
		}
		for nestedKey, nestedValue := range typedValue {
			f.inspect(nestedKey, nestedValue, false)
		}
	case []interface{}:
		for _, nestedValue := range typedValue {
			f.inspect("", nestedValue, false)
		}
	}
}

func (f *sandboxNetNSFinder) inspectScalar(key string, value interface{}, allowPID bool) {
	normalizedKey := normalizeSandboxInfoKey(key)
	if strValue, ok := sandboxInfoString(value); ok {
		if isSandboxNetNSPathKey(normalizedKey) || (normalizedKey == "path" && strings.Contains(strValue, "/ns/net")) {
			f.path = strValue
			return
		}
		if allowPID && isSandboxPIDKey(normalizedKey) {
			if pid, ok := sandboxInfoPID(strValue); ok {
				f.pid = pid
			}
		}
	}
	if allowPID && isSandboxPIDKey(normalizedKey) {
		if pid, ok := sandboxInfoPID(value); ok {
			f.pid = pid
		}
	}
}

func decodeSandboxInfoJSON(value string) (interface{}, bool) {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	var decoded interface{}
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func normalizeSandboxInfoKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, ".", "")
	return key
}

func isSandboxNetNSPathKey(key string) bool {
	switch key {
	case "netnspath",
		"netnamespacepath",
		"networknamespacepath",
		"sandboxnetnspath",
		"sandboxnetnamespacepath":
		return true
	}
	return false
}

func isSandboxPIDKey(key string) bool {
	switch key {
	case "pid", "sandboxpid", "processpid":
		return true
	}
	return false
}

func sandboxInfoString(value interface{}) (string, bool) {
	switch typedValue := value.(type) {
	case string:
		if typedValue == "" {
			return "", false
		}
		return typedValue, true
	case json.Number:
		return typedValue.String(), true
	}
	return "", false
}

func sandboxInfoPID(value interface{}) (int, bool) {
	switch typedValue := value.(type) {
	case int:
		return positivePID(typedValue)
	case int64:
		return positivePID(int(typedValue))
	case float64:
		return positivePID(int(typedValue))
	case json.Number:
		pid, err := typedValue.Int64()
		if err != nil {
			return 0, false
		}
		return positivePID(int(pid))
	case string:
		pid, err := strconv.Atoi(strings.TrimSpace(typedValue))
		if err != nil {
			return 0, false
		}
		return positivePID(pid)
	}
	return 0, false
}

func positivePID(pid int) (int, bool) {
	if pid <= 0 {
		return 0, false
	}
	return pid, true
}

func isNetworkNamespaceEntry(value map[string]interface{}) bool {
	typeValue, ok := mapStringValue(value, "type")
	if !ok {
		return false
	}
	typeValue = strings.ToLower(typeValue)
	return typeValue == "network" || typeValue == "net"
}

func namespaceEntryPath(value map[string]interface{}) (string, bool) {
	pathValue, ok := mapStringValue(value, "path")
	if !ok || pathValue == "" {
		return "", false
	}
	return pathValue, true
}

func mapStringValue(value map[string]interface{}, key string) (string, bool) {
	for candidateKey, candidateValue := range value {
		if normalizeSandboxInfoKey(candidateKey) != normalizeSandboxInfoKey(key) {
			continue
		}
		return sandboxInfoString(candidateValue)
	}
	return "", false
}
