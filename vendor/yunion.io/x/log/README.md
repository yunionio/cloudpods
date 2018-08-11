# log

Log package for go.

Nicely color-format-coded in development (when a TTY is attached, otherwise just plain text):

![Colored](https://i.imgur.com/aMduIBK.png)

### Example

```go
package main

import (
  "errors"

  "yunion.io/x/log"
)

func main() {
  log.Debugf("Debug msg")
  log.Infof("Info msg")
  log.Warningf("Warning msg")
  err := errors.New("Unknown error")
  log.Errorf("Error msg: %v", err)
}
```
