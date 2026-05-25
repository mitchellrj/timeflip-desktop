//go:build darwin

package device

import (
	"context"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/tinygo-org/cbgo"
)

func defaultBluetoothAvailabilityCheck(context.Context) error {
	manager := cbgo.NewCentralManager(&cbgo.ManagerOpts{ShowPowerAlert: false})
	switch manager.State() {
	case cbgo.ManagerStatePoweredOn:
		return nil
	case cbgo.ManagerStatePoweredOff:
		return bluetoothUnavailable("Bluetooth is turned off.", "CoreBluetooth manager state is powered off")
	case cbgo.ManagerStateUnauthorized:
		return bluetoothPermissionDenied("Bluetooth permission is not available for TimeFlip Desktop.", "CoreBluetooth manager state is unauthorized")
	case cbgo.ManagerStateUnsupported:
		return bluetoothUnavailable("Bluetooth is not supported on this Mac.", "CoreBluetooth manager state is unsupported")
	default:
		return nil
	}
}

func bluetoothUnavailable(message string, diagnostic string) error {
	return domain.DeviceWorkflowError{AppError: domain.NewAppError(domain.ErrBluetoothUnavailable, message, diagnostic, nil)}
}

func bluetoothPermissionDenied(message string, diagnostic string) error {
	return domain.DeviceWorkflowError{AppError: domain.NewAppError(domain.ErrBluetoothPermissionDenied, message, diagnostic, nil)}
}
