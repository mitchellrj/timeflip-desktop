package device

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	timeflip "github.com/mitchellrj/timeflip-go"
)

func TestTracingReadSuppressesOptionalDeviceNameProtocolError(t *testing.T) {
	var out bytes.Buffer
	conn := tracingConnection{
		inner:    traceReadConnection{err: timeflip.ErrProtocol},
		deviceID: "d1",
		log:      &traceLogger{out: &out},
	}

	_, err := conn.Read(context.Background(), timeflip.CharacteristicID("0x2A00"))
	if !errors.Is(err, timeflip.ErrProtocol) {
		t.Fatalf("expected original protocol error to be returned, got %v", err)
	}
	if strings.Contains(out.String(), "error=") {
		t.Fatalf("expected optional device-name miss to be logged without error, got %s", out.String())
	}
}

func TestTracingReadKeepsUnexpectedReadErrors(t *testing.T) {
	var out bytes.Buffer
	conn := tracingConnection{
		inner:    traceReadConnection{err: timeflip.ErrDisconnected},
		deviceID: "d1",
		log:      &traceLogger{out: &out},
	}

	_, _ = conn.Read(context.Background(), timeflip.CharacteristicID("0x2A19"))
	if !strings.Contains(out.String(), "error=") {
		t.Fatalf("expected non-optional read error to remain in trace, got %s", out.String())
	}
}

type traceReadConnection struct {
	err error
}

func (c traceReadConnection) Read(context.Context, timeflip.CharacteristicID) ([]byte, error) {
	return nil, c.err
}

func (c traceReadConnection) Write(context.Context, timeflip.CharacteristicID, []byte) error {
	return nil
}

func (c traceReadConnection) Subscribe(context.Context, timeflip.CharacteristicID) (<-chan timeflip.Notification, error) {
	return nil, nil
}

func (c traceReadConnection) Close(context.Context) error {
	return nil
}
