package exec

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

type testHandler struct {
	conn    net.Conn
	handler ProcessHandler
	t       *testing.T
}

func newTestHandler(t *testing.T) Executor {
	handler := func(command string, args []string, f *os.File) (ProcessHandler, error) {
		// Spawn Plugin
		conn, err := net.FileConn(f)
		if err != nil {
			return nil, err
		}
		plugin, err := RunPlugin(conn)
		if err != nil {
			return nil, err
		}

		return &testHandler{conn, plugin, t}, nil
	}
	return handler
}

func (h *testHandler) Kill() error {
	h.t.Logf("Killing handler")
	return h.handler.Kill() //h.conn.Close()
}

func (h *testHandler) Wait() error {
	h.t.Logf("Waiting")
	return h.handler.Wait() //<-h.errChan
}

func limitTime(f func() error, d time.Duration) error {
	retChan := make(chan error)
	timer := time.After(d)
	go func() {
		retChan <- f()
	}()

	select {
	case err := <-retChan:
		return err
	case <-timer:
		return errors.New("time limit exceeded")
	}

}

func TestConnection(t *testing.T) {
	sub, err := Spawn("test", nil, newTestHandler(t))
	if err != nil {
		t.Fatalf("Error spawning test: %s", err)
	}

	if err := limitTime(sub.Close, 3*time.Second); err != nil {
		t.Fatalf("Error closing test: %s", err)
	}
}
