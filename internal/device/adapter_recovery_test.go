package device

import (
	"context"
	"errors"
	"testing"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	timeflip "github.com/mitchellrj/timeflip-go"
)

func TestAdapterRecoveryClientRebuildsAndRetriesStuckEnable(t *testing.T) {
	first := &recordingClient{connectErr: errors.New("macos_adapter: already calling Enable function")}
	second := &recordingClient{}
	builds := 0
	client, err := newAdapterRecoveryClient(func() (Client, error) {
		builds++
		if builds == 1 {
			return first, nil
		}
		return second, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	handle, err := client.Connect(context.Background(), ConnectRequest{DeviceID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if handle.DeviceID() != "d1" {
		t.Fatalf("unexpected handle %q", handle.DeviceID())
	}
	if builds != 2 {
		t.Fatalf("expected client rebuild, got %d builds", builds)
	}
	if first.connects != 1 || second.connects != 1 {
		t.Fatalf("expected one connect on each client, got first=%d second=%d", first.connects, second.connects)
	}
}

func TestAdapterRecoveryClientRebuildsButDoesNotRetryEnableTimeout(t *testing.T) {
	first := &recordingClient{connectErr: errors.New("macos_adapter: timeout enabling CentralManager")}
	second := &recordingClient{}
	builds := 0
	client, err := newAdapterRecoveryClient(func() (Client, error) {
		builds++
		if builds == 1 {
			return first, nil
		}
		return second, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.Connect(context.Background(), ConnectRequest{DeviceID: "d1"}); err == nil {
		t.Fatal("expected connect error")
	}
	if builds != 2 {
		t.Fatalf("expected client rebuild for next attempt, got %d builds", builds)
	}
	if first.connects != 1 || second.connects != 0 {
		t.Fatalf("unexpected connect attempts first=%d second=%d", first.connects, second.connects)
	}
}

func TestMapDeviceErrorClassifiesAdapterEnableFailures(t *testing.T) {
	err := MapDeviceError(errors.New("connect: macos_adapter: already calling Enable function"))
	var workflowErr domain.DeviceWorkflowError
	if !errors.As(err, &workflowErr) {
		t.Fatalf("expected workflow error, got %T", err)
	}
	if workflowErr.Code != domain.ErrBluetoothUnavailable {
		t.Fatalf("expected bluetooth unavailable, got %s", workflowErr.Code)
	}
	if workflowErr.Message == "" || workflowErr.Message == "Device operation failed." {
		t.Fatalf("expected actionable message, got %q", workflowErr.Message)
	}
}

func TestIsEventDecodeErrorClassifiesProtocolEventErrors(t *testing.T) {
	err := &timeflip.OperationError{Operation: "events", Stage: "history", Err: timeflip.ErrProtocol}
	if !IsEventDecodeError(err) {
		t.Fatalf("expected event protocol error to be classified as decode error")
	}
	err = &timeflip.OperationError{Operation: "events", Stage: "history", Err: timeflip.ErrDisconnected}
	if IsEventDecodeError(err) {
		t.Fatalf("disconnect should not be classified as decode error")
	}
}
