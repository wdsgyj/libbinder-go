package libbinder

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRPCTCPTransportHelpers(t *testing.T) {
	listener, err := ListenRPCTCP("127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenRPCTCP: %v", err)
	}
	defer func() { _ = listener.Close() }()

	acceptCh := acceptRPCAsync(listener, RPCConfig{})
	client, err := DialRPCTCP(listener.Addr().String(), RPCConfig{})
	if err != nil {
		t.Fatalf("DialRPCTCP: %v", err)
	}
	server := awaitRPCConn(t, acceptCh)
	defer func() {
		_ = client.Close()
		_ = server.Close()
	}()

	assertRPCServiceRoundTrip(t, server, client)
}

func TestRPCUnixTransportHelpers(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("android test sandbox does not permit binding unix sockets under the test temp dir")
	}
	path := filepath.Join(t.TempDir(), "rpc.sock")
	listener, err := ListenRPCUnix(path)
	if err != nil {
		t.Fatalf("ListenRPCUnix: %v", err)
	}
	defer func() { _ = listener.Close() }()

	acceptCh := acceptRPCAsync(listener, RPCConfig{})
	client, err := DialRPCUnix(path, RPCConfig{})
	if err != nil {
		t.Fatalf("DialRPCUnix: %v", err)
	}
	server := awaitRPCConn(t, acceptCh)
	defer func() {
		_ = client.Close()
		_ = server.Close()
	}()

	assertRPCServiceRoundTrip(t, server, client)
}

func TestRPCTLSTransportHelpers(t *testing.T) {
	serverTLS, clientTLS := newRPCTransportTLSConfigs(t)
	listener, err := ListenRPCTLS("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatalf("ListenRPCTLS: %v", err)
	}
	defer func() { _ = listener.Close() }()

	acceptCh := acceptRPCAsync(listener, RPCConfig{})
	client, err := DialRPCTLS("tcp", listener.Addr().String(), clientTLS, RPCConfig{})
	if err != nil {
		t.Fatalf("DialRPCTLS: %v", err)
	}
	server := awaitRPCConn(t, acceptCh)
	defer func() {
		_ = client.Close()
		_ = server.Close()
	}()

	assertRPCServiceRoundTrip(t, server, client)
}

func TestRPCWatchDeathOnServerClose(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server, err := ServeRPC(serverConn)
	if err != nil {
		t.Fatalf("ServeRPC: %v", err)
	}
	client, err := DialRPC(clientConn)
	if err != nil {
		t.Fatalf("DialRPC: %v", err)
	}

	if err := server.ServiceManager().AddService(context.Background(), "echo", api.StaticHandler{
		DescriptorName: "rpc.echo",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			reply := api.NewParcel()
			if err := reply.WriteString("ok"); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}); err != nil {
		t.Fatalf("server AddService: %v", err)
	}

	service, err := client.ServiceManager().WaitService(context.Background(), "echo")
	if err != nil {
		t.Fatalf("WaitService: %v", err)
	}
	sub, err := service.WatchDeath(context.Background())
	if err != nil {
		t.Fatalf("WatchDeath: %v", err)
	}

	if err := server.Close(); err != nil {
		t.Fatalf("server.Close: %v", err)
	}
	defer func() { _ = client.Close() }()

	select {
	case <-sub.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("WatchDeath did not complete")
	}
	if sub.Err() != api.ErrDeadObject {
		t.Fatalf("WatchDeath.Err = %v, want %v", sub.Err(), api.ErrDeadObject)
	}
}

type rpcAcceptResult struct {
	conn *RPCConn
	err  error
}

func acceptRPCAsync(listener net.Listener, cfg RPCConfig) <-chan rpcAcceptResult {
	ch := make(chan rpcAcceptResult, 1)
	go func() {
		conn, err := AcceptRPC(listener, cfg)
		ch <- rpcAcceptResult{conn: conn, err: err}
	}()
	return ch
}

func awaitRPCConn(t *testing.T, ch <-chan rpcAcceptResult) *RPCConn {
	t.Helper()
	select {
	case result := <-ch:
		if result.err != nil {
			t.Fatalf("AcceptRPC: %v", result.err)
		}
		return result.conn
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for AcceptRPC")
		return nil
	}
}

func assertRPCServiceRoundTrip(t *testing.T, server, client *RPCConn) {
	t.Helper()

	if err := server.ServiceManager().AddService(context.Background(), "echo", api.StaticHandler{
		DescriptorName: "rpc.echo",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			msg, err := data.ReadString()
			if err != nil {
				return nil, err
			}
			reply := api.NewParcel()
			if err := reply.WriteString("echo:" + msg); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}); err != nil {
		t.Fatalf("AddService: %v", err)
	}

	service, err := client.ServiceManager().WaitService(context.Background(), "echo")
	if err != nil {
		t.Fatalf("WaitService: %v", err)
	}
	req := api.NewParcel()
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	reply, err := service.Transact(context.Background(), api.FirstCallTransaction, req, api.FlagNone)
	if err != nil {
		t.Fatalf("Transact: %v", err)
	}
	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != "echo:ping" {
		t.Fatalf("reply = %q, want echo:ping", got)
	}
}

func newRPCTransportTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "libbinder-go-rpc-test",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, &tls.Config{InsecureSkipVerify: true}
}
