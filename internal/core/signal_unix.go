package core

import (
	"os"
	"syscall"
)

// interruptSignal returns the signal used to ask a core to shut down gracefully.
func interruptSignal() os.Signal { return syscall.SIGTERM }
