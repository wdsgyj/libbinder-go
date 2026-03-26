package libbinder

import (
	"crypto/tls"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func DialRPCNetwork(network, address string, cfg RPCConfig) (*RPCConn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	rpcConn, err := DialRPCWithConfig(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return rpcConn, nil
}

func DialRPCTCP(address string, cfg RPCConfig) (*RPCConn, error) {
	return DialRPCNetwork("tcp", address, cfg)
}

func DialRPCUnix(address string, cfg RPCConfig) (*RPCConn, error) {
	return DialRPCNetwork("unix", address, cfg)
}

func DialRPCTLS(network, address string, tlsConfig *tls.Config, cfg RPCConfig) (*RPCConn, error) {
	conn, err := tls.Dial(network, address, tlsConfig)
	if err != nil {
		return nil, err
	}
	rpcConn, err := DialRPCWithConfig(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return rpcConn, nil
}

func ListenRPC(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func ListenRPCTCP(address string) (net.Listener, error) {
	return ListenRPC("tcp", address)
}

func ListenRPCUnix(address string) (net.Listener, error) {
	address, cleanup, err := rpcPrepareUnixListenAddress(address)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("unix", address)
	if err != nil {
		if cleanup != nil {
			_ = cleanup()
		}
		return nil, err
	}
	if cleanup == nil {
		return listener, nil
	}
	return &rpcCleanupListener{
		Listener: listener,
		cleanup:  cleanup,
	}, nil
}

func ListenRPCTLS(network, address string, tlsConfig *tls.Config) (net.Listener, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(listener, tlsConfig), nil
}

func AcceptRPC(listener net.Listener, cfg RPCConfig) (*RPCConn, error) {
	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}
	rpcConn, err := ServeRPCWithConfig(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return rpcConn, nil
}

type rpcCleanupListener struct {
	net.Listener
	cleanup func() error
	once    sync.Once
}

func (l *rpcCleanupListener) Close() error {
	closeErr := l.Listener.Close()
	var cleanupErr error
	l.once.Do(func() {
		if l.cleanup != nil {
			cleanupErr = l.cleanup()
		}
	})
	return errors.Join(closeErr, cleanupErr)
}

func rpcPrepareUnixListenAddress(address string) (string, func() error, error) {
	if address != "" {
		return address, nil, nil
	}
	if runtime.GOOS == "android" {
		return rpcAbstractUnixAddress(), nil, nil
	}
	dir, err := os.MkdirTemp("", "libbinder-go-rpc-*")
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(dir, "rpc.sock"), func() error {
		return os.RemoveAll(dir)
	}, nil
}

func rpcAbstractUnixAddress() string {
	return "@libbinder-go-rpc-" + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
