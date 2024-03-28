package remotecommand

// TerminalSize and TerminalSizeQueue was a port of k8s.io/kubernetes/pkg/util/term
// and were moved in order to decouple client from other term dependencies

// TerminalSize represents the width and height of a terminal
type TerminalSize struct {
	Width  uint16
	Height uint16
}

// TerminalSizeQueue is capable of returning terminal resize events as they occur.
type TerminalSizeQueue interface {
	// Next returns the new terminal size after the terminal has been resized. It returns nil when
	// monitoring has been stopped.
	Next() *TerminalSize
}
