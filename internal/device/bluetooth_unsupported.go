//go:build !darwin

package device

import "context"

func defaultBluetoothAvailabilityCheck(context.Context) error {
	return nil
}
