package libbinder

import (
	"crypto/tls"
	"net"
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
	return ListenRPC("unix", address)
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
