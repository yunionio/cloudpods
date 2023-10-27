package bingocloud

import (
	"fmt"
	"strings"
)

func nextDeviceName(curDeviceNames []string) (string, error) {
	var currents []string
	for _, item := range curDeviceNames {
		currents = append(currents, strings.ToLower(item))
	}

	for i := 0; i < 25; i++ {
		device := fmt.Sprintf("/dev/vd%c", byte(98+i))
		found := false
		for _, item := range currents {
			if strings.HasPrefix(item, device) {
				found = true
			}
		}

		if !found {
			return device, nil
		}
	}

	return "", fmt.Errorf("disk devicename out of index, current deivces: %s", currents)
}
