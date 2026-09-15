package conffile

import (
	"os"
	"os/user"
)

func homedir() string {
	home := os.Getenv(homeEnv)
	if home == "" {
		u, err := user.Current()
		if err == nil {
			home = u.HomeDir
		}
	}
	return home
}
