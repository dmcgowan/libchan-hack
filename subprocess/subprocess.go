package exec

import (
	"fmt"
	"io"
	"net"
	"os"
	"syscall"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
)

type SubProcess struct {
	conn      net.Conn
	handler   ProcessHandler
	transport libchan.Transport
}

type Executor func(command string, args []string, f *os.File) (ProcessHandler, error)

type ProcessHandler interface {
	Kill() error
	Wait() error
}

type subProcessHandler struct {
	process *os.Process
}

func NewSubProcessHandler(command string, args []string, f *os.File) (ProcessHandler, error) {
	var attr os.ProcAttr
	attr.Files = append(attr.Files, f)
	attr.Env = append(attr.Env, fmt.Sprintf("LIBCHAN_FD=%d", f.Fd()))

	proc, err := os.StartProcess(command, args, &attr)
	if err != nil {
		return nil, err
	}

	return &subProcessHandler{proc}, nil
}

type SubProcessError struct {
	State *os.ProcessState
}

func (err SubProcessError) Error() string {
	return fmt.Sprintf("process error: %s", err.State.String())
}

func (h *subProcessHandler) Kill() error {
	return h.process.Kill()
}
func (h *subProcessHandler) Wait() error {
	state, err := h.process.Wait()
	if err != nil {
		return err
	}

	if !state.Success() {
		return SubProcessError{state}
	}

	return nil

}

func Spawn(command string, args []string, executor Executor) (*SubProcess, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	handler, err := executor(command, args, os.NewFile(uintptr(fds[1]), ""))
	if err != nil {
		return nil, err
	}

	conn, err := net.FileConn(os.NewFile(uintptr(fds[0]), ""))
	if err != nil {
		return nil, err
	}

	t, err := spdy.NewClientTransport(conn)
	if err != nil {
		return nil, err
	}

	sender, err := t.NewSendChannel()
	if err != nil {
		return nil, err
	}

	recv, s := libchan.Pipe()
	if err := sender.Send(&EchoMessage{"hello", s}); err != nil {
		return nil, err
	}
	var echo EchoMessage
	if err := recv.Receive(&echo); err != nil {
		return nil, err
	}

	return &SubProcess{conn, handler, t}, nil
}

func (s *SubProcess) CreateChannel() (libchan.Sender, error) {
	return s.transport.NewSendChannel()
}

func (s *SubProcess) Close() error {
	if err := s.handler.Kill(); err != nil {
		return err
	}
	s.conn.Close()
	return s.handler.Wait()
}

type plugin struct {
	transport io.Closer
	errChan   chan error
}

func (p *plugin) Kill() error {
	return p.transport.Close()
}

func (p *plugin) Wait() error {
	return <-p.errChan
}

type EchoMessage struct {
	Message string
	Ret     libchan.Sender
}

func RunPlugin(conn net.Conn) (ProcessHandler, error) {
	transport, err := spdy.NewServerTransport(conn)
	if err != nil {
		return nil, err
	}

	retChan := make(chan error)
	go func() {
		receiver, err := transport.WaitReceiveChannel()
		if err != nil {
			retChan <- err
			return
		}

		var echo EchoMessage
		if err := receiver.Receive(&echo); err != nil {
			retChan <- err
			return
		}
		if err := echo.Ret.Send(&EchoMessage{Message: echo.Message}); err != nil {
			retChan <- err
			return
		}

		// Start loop
		//for {
		//	_, err := transport.WaitReceiveChannel()
		//	if err != nil {
		//		retChan <- err
		//		break
		//	}

		//	// Do something with recv
		//}
		close(retChan)
	}()
	return &plugin{transport, retChan}, nil
}
