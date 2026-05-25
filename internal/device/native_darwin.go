//go:build darwin

package device

import (
	"time"

	"github.com/mitchellrj/timeflip-go/macos"
)

func NewNativeDeviceClient(timeout time.Duration) (Client, error) {
	return NewTimeflipDeviceClient(macos.NewTransport(), timeout)
}
