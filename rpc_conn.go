package libbinder

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type rpcRole uint8

const (
	rpcRoleClient rpcRole = iota + 1
	rpcRoleServer
)

const (
	rpcServerServiceManagerHandle = 1
	rpcClientExportStart          = 2
	rpcServerExportStart          = 3
)

type rpcFrameKind string

const (
	rpcFrameTransact rpcFrameKind = "transact"
	rpcFrameReply    rpcFrameKind = "reply"
	rpcFrameObituary rpcFrameKind = "obituary"
)

type rpcFrame struct {
	Kind    rpcFrameKind
	ID      uint64
	Handle  uint32
	Code    uint32
	Flags   api.Flags
	Payload []byte
	Objects []api.ParcelObject
	Err     *rpcFrameError
}

type rpcFrameError struct {
	Kind          string
	Message       string
	StatusCode    int32
	ExceptionCode api.ExceptionCode
}

type rpcExport struct {
	handle    uint32
	handler   api.Handler
	stability api.StabilityLevel
	serial    bool
	mu        sync.Mutex
}

type RPCDebugSnapshot struct {
	Closed          bool
	ExportedObjects int
	ImportedObjects int
	PendingCalls    int
	FramePoolGets   uint64
	FramePoolPuts   uint64
	ServiceManager  ServiceManagerSnapshot
}

type RPCConfig struct {
	RequiredStability api.StabilityLevel
}

type RPCConn struct {
	conn              net.Conn
	role              rpcRole
	requiredStability api.StabilityLevel

	enc     *gob.Encoder
	dec     *gob.Decoder
	writeMu sync.Mutex

	closeOnce sync.Once
	closed    chan struct{}

	errMu    sync.Mutex
	closeErr error

	nextRequest atomic.Uint64

	pendingMu sync.Mutex
	pending   map[uint64]chan rpcFrame

	exportsMu      sync.RWMutex
	exports        map[uint32]*rpcExport
	imports        map[uint32]*rpcRemoteBinder
	nextExportID   uint32
	localRoleStart uint32

	serviceRegistry       *rpcServiceRegistry
	serviceManagerHandler *rpcServiceManagerHandler
	serviceManager        interface {
		api.ServiceManager
		debugSnapshot() serviceManagerDebugSnapshot
	}

	framePool sync.Pool
	frameGets atomic.Uint64
	framePuts atomic.Uint64

	deathMu      sync.Mutex
	deathWatches map[uint32]map[*managedSubscription]struct{}
}

func DialRPC(conn net.Conn) (*RPCConn, error) {
	return DialRPCWithConfig(conn, RPCConfig{})
}

func ServeRPC(conn net.Conn) (*RPCConn, error) {
	return ServeRPCWithConfig(conn, RPCConfig{})
}

func DialRPCWithConfig(conn net.Conn, cfg RPCConfig) (*RPCConn, error) {
	return newRPCConn(conn, rpcRoleClient, cfg)
}

func ServeRPCWithConfig(conn net.Conn, cfg RPCConfig) (*RPCConn, error) {
	return newRPCConn(conn, rpcRoleServer, cfg)
}

func newRPCConn(conn net.Conn, role rpcRole, cfg RPCConfig) (*RPCConn, error) {
	if conn == nil {
		return nil, api.ErrUnsupported
	}

	c := &RPCConn{
		conn:              conn,
		role:              role,
		requiredStability: cfg.RequiredStability,
		enc:               gob.NewEncoder(conn),
		dec:               gob.NewDecoder(conn),
		closed:            make(chan struct{}),
		pending:           make(map[uint64]chan rpcFrame),
		exports:           make(map[uint32]*rpcExport),
		imports:           make(map[uint32]*rpcRemoteBinder),
		serviceRegistry:   newRPCServiceRegistry(),
		deathWatches:      make(map[uint32]map[*managedSubscription]struct{}),
	}
	if c.requiredStability == api.StabilityUndeclared {
		c.requiredStability = api.DefaultLocalStability()
	}
	c.framePool.New = func() any {
		return &rpcFrame{}
	}

	switch role {
	case rpcRoleServer:
		c.localRoleStart = 1
		c.nextExportID = rpcServerExportStart
		c.serviceManagerHandler = newRPCServiceManagerHandler(c)
		c.exportWithHandle(rpcServerServiceManagerHandle, c.serviceManagerHandler, api.DefaultLocalStability(), false)
		c.serviceManager = &rpcLocalServiceManager{conn: c}
	case rpcRoleClient:
		c.localRoleStart = 2
		c.nextExportID = rpcClientExportStart
		c.serviceManager = &rpcRemoteServiceManager{
			conn:   c,
			target: newRPCRemoteBinder(c, rpcServerServiceManagerHandle, api.DefaultLocalStability()),
		}
	default:
		return nil, api.ErrUnsupported
	}

	go c.readLoop()
	return c, nil
}

func (c *RPCConn) Close() error {
	if c == nil {
		return nil
	}
	c.shutdown(io.EOF)
	c.errMu.Lock()
	defer c.errMu.Unlock()
	if c.closeErr == nil || errors.Is(c.closeErr, io.EOF) || errors.Is(c.closeErr, net.ErrClosed) {
		return nil
	}
	return c.closeErr
}

func (c *RPCConn) ServiceManager() api.ServiceManager {
	if c == nil {
		return nil
	}
	return c.serviceManager
}

func (c *RPCConn) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	return c.registerLocalHandler(handler, false)
}

func (c *RPCConn) DebugSnapshot() RPCDebugSnapshot {
	if c == nil {
		return RPCDebugSnapshot{}
	}

	out := RPCDebugSnapshot{
		FramePoolGets: c.frameGets.Load(),
		FramePoolPuts: c.framePuts.Load(),
	}
	select {
	case <-c.closed:
		out.Closed = true
	default:
	}

	c.exportsMu.RLock()
	out.ExportedObjects = len(c.exports)
	out.ImportedObjects = len(c.imports)
	c.exportsMu.RUnlock()

	c.pendingMu.Lock()
	out.PendingCalls = len(c.pending)
	c.pendingMu.Unlock()

	if c.serviceManager != nil {
		sm := c.serviceManager.debugSnapshot()
		out.ServiceManager = ServiceManagerSnapshot{
			CacheEntries: sm.CacheEntries,
			CacheHits:    sm.CacheHits,
			CacheMisses:  sm.CacheMisses,
			Names:        sm.Names,
		}
	}
	return out
}

func (c *RPCConn) registerLocalHandler(handler api.Handler, serial bool) (api.Binder, error) {
	if c == nil || handler == nil {
		return nil, api.ErrUnsupported
	}

	c.exportsMu.Lock()
	handle := c.nextExportID
	c.nextExportID += 2
	export := &rpcExport{
		handle:    handle,
		handler:   handler,
		stability: api.HandlerStability(handler),
		serial:    serial,
	}
	c.exports[handle] = export
	c.exportsMu.Unlock()
	return newRPCLocalBinder(c, handle, export.stability), nil
}

func (c *RPCConn) unregisterLocalHandle(handle uint32) error {
	if c == nil || handle == 0 || handle == rpcServerServiceManagerHandle {
		return nil
	}
	c.exportsMu.Lock()
	delete(c.exports, handle)
	c.exportsMu.Unlock()
	frame := c.getFrame()
	frame.Kind = rpcFrameObituary
	frame.Handle = handle
	err := c.writeFrame(frame)
	c.putFrame(frame)
	return err
}

func (c *RPCConn) exportWithHandle(handle uint32, handler api.Handler, stability api.StabilityLevel, serial bool) {
	c.exportsMu.Lock()
	defer c.exportsMu.Unlock()
	c.exports[handle] = &rpcExport{
		handle:    handle,
		handler:   handler,
		stability: stability,
		serial:    serial,
	}
}

func (c *RPCConn) lookupExport(handle uint32) (*rpcExport, bool) {
	if c == nil {
		return nil, false
	}
	c.exportsMu.RLock()
	defer c.exportsMu.RUnlock()
	export, ok := c.exports[handle]
	return export, ok
}

func (c *RPCConn) remoteBinder(handle uint32, stability api.StabilityLevel) *rpcRemoteBinder {
	c.exportsMu.Lock()
	defer c.exportsMu.Unlock()

	if remote := c.imports[handle]; remote != nil {
		if stability != api.StabilityUndeclared {
			remote.stability = stability
		}
		return remote
	}

	remote := newRPCRemoteBinder(c, handle, stability)
	c.imports[handle] = remote
	return remote
}

func (c *RPCConn) resolveRPCObject(obj api.ParcelObject) api.Binder {
	if export, ok := c.lookupExport(obj.Handle); ok {
		return newRPCLocalBinder(c, export.handle, export.stability)
	}
	return c.remoteBinder(obj.Handle, obj.Stability)
}

func (c *RPCConn) dispatchExport(ctx context.Context, handle uint32, dataCode uint32, data *api.Parcel, flags api.Flags, local bool) (*api.Parcel, error) {
	export, ok := c.lookupExport(handle)
	if !ok {
		return nil, api.ErrNoService
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if data == nil {
		data = api.NewParcel()
	}
	data.SetBinderObjectResolvers(c.resolveRPCObject, nil)

	var serial sync.Locker
	if export.serial {
		serial = &export.mu
	}
	return api.DispatchLocalHandler(ctx, export.handler, serial, dataCode, data, flags, api.TransactionContext{
		Code:  dataCode,
		Flags: flags,
		Local: local,
	})
}

func (c *RPCConn) transactRemote(ctx context.Context, handle uint32, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if c == nil {
		return nil, api.ErrUnsupported
	}
	select {
	case <-c.closed:
		return nil, api.ErrClosed
	default:
	}

	payload, objects, err := c.prepareRPCParcel(data)
	if err != nil {
		return nil, err
	}

	frame := c.getFrame()
	frame.Kind = rpcFrameTransact
	frame.ID = c.nextRequest.Add(1)
	frame.Handle = handle
	frame.Code = code
	frame.Flags = flags
	frame.Payload = payload
	frame.Objects = objects

	var pending chan rpcFrame
	if flags&api.FlagOneway == 0 {
		pending = make(chan rpcFrame, 1)
		c.pendingMu.Lock()
		c.pending[frame.ID] = pending
		c.pendingMu.Unlock()
	}

	if err := c.writeFrame(frame); err != nil {
		c.clearPending(frame.ID)
		c.putFrame(frame)
		return nil, err
	}
	c.putFrame(frame)

	if flags&api.FlagOneway != 0 {
		return nil, nil
	}

	select {
	case reply, ok := <-pending:
		if !ok {
			return nil, c.connectionError()
		}
		if reply.Err != nil {
			return nil, reply.Err.replay()
		}
		parcel := api.NewParcelWire(reply.Payload, reply.Objects)
		parcel.SetBinderObjectResolvers(c.resolveRPCObject, nil)
		return parcel, nil
	case <-ctx.Done():
		c.clearPending(frame.ID)
		return nil, ctx.Err()
	case <-c.closed:
		c.clearPending(frame.ID)
		return nil, c.connectionError()
	}
}

func (c *RPCConn) prepareRPCParcel(parcel *api.Parcel) ([]byte, []api.ParcelObject, error) {
	if parcel == nil {
		return nil, nil, nil
	}

	payload := parcel.Bytes()
	objects := parcel.Objects()
	for _, obj := range objects {
		switch obj.Kind {
		case api.ObjectFileDescriptor:
			return nil, nil, fmt.Errorf("%w: file descriptors are not supported over RPC", api.ErrUnsupported)
		case api.ObjectStrongBinder:
			if obj.Handle == 0 {
				return nil, nil, fmt.Errorf("%w: kernel-local binder objects are not supported over RPC", api.ErrUnsupported)
			}
			if _, ok := c.lookupExport(obj.Handle); ok {
				continue
			}
			c.exportsMu.RLock()
			_, ok := c.imports[obj.Handle]
			c.exportsMu.RUnlock()
			if !ok {
				return nil, nil, fmt.Errorf("%w: binder handle %d does not belong to this RPC session", api.ErrUnsupported, obj.Handle)
			}
		}
	}
	return payload, objects, nil
}

func (c *RPCConn) readLoop() {
	for {
		var frame rpcFrame
		if err := c.dec.Decode(&frame); err != nil {
			c.shutdown(err)
			return
		}

		switch frame.Kind {
		case rpcFrameReply:
			c.pendingMu.Lock()
			ch := c.pending[frame.ID]
			delete(c.pending, frame.ID)
			c.pendingMu.Unlock()
			if ch != nil {
				ch <- frame
				close(ch)
			}
		case rpcFrameTransact:
			go c.handleTransactFrame(frame)
		case rpcFrameObituary:
			c.noteRemoteDeath(frame.Handle)
		default:
			c.shutdown(fmt.Errorf("%w: unknown rpc frame kind %q", api.ErrBadParcelable, frame.Kind))
			return
		}
	}
}

func (c *RPCConn) handleTransactFrame(frame rpcFrame) {
	request := api.NewParcelWire(frame.Payload, frame.Objects)
	request.SetBinderObjectResolvers(c.resolveRPCObject, nil)

	reply, err := c.dispatchExport(context.Background(), frame.Handle, frame.Code, request, frame.Flags, false)
	if frame.Flags&api.FlagOneway != 0 {
		return
	}

	replyFrame := c.getFrame()
	replyFrame.Kind = rpcFrameReply
	replyFrame.ID = frame.ID
	if err != nil {
		replyFrame.Err = captureRPCFrameError(err)
	} else {
		replyFrame.Payload, replyFrame.Objects, err = c.prepareRPCParcel(reply)
		if err != nil {
			replyFrame.Err = captureRPCFrameError(err)
		}
	}

	_ = c.writeFrame(replyFrame)
	c.putFrame(replyFrame)
}

func (c *RPCConn) writeFrame(frame *rpcFrame) error {
	if c == nil {
		return api.ErrUnsupported
	}
	select {
	case <-c.closed:
		return c.connectionError()
	default:
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.enc.Encode(frame); err != nil {
		c.shutdown(err)
		return c.connectionError()
	}
	return nil
}

func (c *RPCConn) getFrame() *rpcFrame {
	c.frameGets.Add(1)
	return c.framePool.Get().(*rpcFrame)
}

func (c *RPCConn) putFrame(frame *rpcFrame) {
	if frame == nil {
		return
	}
	*frame = rpcFrame{}
	c.framePuts.Add(1)
	c.framePool.Put(frame)
}

func (c *RPCConn) clearPending(id uint64) {
	if id == 0 {
		return
	}
	c.pendingMu.Lock()
	ch := c.pending[id]
	delete(c.pending, id)
	c.pendingMu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (c *RPCConn) shutdown(err error) {
	c.closeOnce.Do(func() {
		c.noteAllRemoteDeaths()
		if c.serviceRegistry != nil {
			c.serviceRegistry.clearClientState()
		}
		if c.serviceManagerHandler != nil {
			c.serviceManagerHandler.closeAllRemoteSubscriptions()
		}
		c.errMu.Lock()
		c.closeErr = err
		c.errMu.Unlock()

		_ = c.conn.Close()
		close(c.closed)

		c.pendingMu.Lock()
		pending := c.pending
		c.pending = make(map[uint64]chan rpcFrame)
		c.pendingMu.Unlock()
		for _, ch := range pending {
			close(ch)
		}
	})
}

func (c *RPCConn) connectionError() error {
	if c == nil {
		return api.ErrClosed
	}
	c.errMu.Lock()
	defer c.errMu.Unlock()
	if c.closeErr == nil || errors.Is(c.closeErr, io.EOF) || errors.Is(c.closeErr, net.ErrClosed) {
		return api.ErrClosed
	}
	return c.closeErr
}

func captureRPCFrameError(err error) *rpcFrameError {
	if err == nil {
		return nil
	}

	var statusErr *api.StatusCodeError
	if errors.As(err, &statusErr) {
		return &rpcFrameError{
			Kind:       "status_code",
			StatusCode: statusErr.Code,
		}
	}

	var remoteErr *api.RemoteException
	if errors.As(err, &remoteErr) {
		return &rpcFrameError{
			Kind:          "remote_exception",
			Message:       remoteErr.Message,
			ExceptionCode: remoteErr.Code,
		}
	}

	switch {
	case errors.Is(err, api.ErrDeadObject):
		return &rpcFrameError{Kind: "err_dead_object"}
	case errors.Is(err, api.ErrFailedTxn):
		return &rpcFrameError{Kind: "err_failed_txn"}
	case errors.Is(err, api.ErrBadParcelable):
		return &rpcFrameError{Kind: "err_bad_parcelable"}
	case errors.Is(err, api.ErrPermissionDenied):
		return &rpcFrameError{Kind: "err_permission_denied"}
	case errors.Is(err, api.ErrUnsupported):
		return &rpcFrameError{Kind: "err_unsupported"}
	case errors.Is(err, api.ErrNoService):
		return &rpcFrameError{Kind: "err_no_service"}
	case errors.Is(err, api.ErrClosed):
		return &rpcFrameError{Kind: "err_closed"}
	case errors.Is(err, api.ErrUnknownTransaction):
		return &rpcFrameError{Kind: "err_unknown_transaction"}
	default:
		return &rpcFrameError{
			Kind:    "text",
			Message: err.Error(),
		}
	}
}

func (e *rpcFrameError) replay() error {
	if e == nil {
		return nil
	}

	switch e.Kind {
	case "status_code":
		return &api.StatusCodeError{Code: e.StatusCode}
	case "remote_exception":
		return &api.RemoteException{Code: e.ExceptionCode, Message: e.Message}
	case "err_dead_object":
		return api.ErrDeadObject
	case "err_failed_txn":
		return api.ErrFailedTxn
	case "err_bad_parcelable":
		return api.ErrBadParcelable
	case "err_permission_denied":
		return api.ErrPermissionDenied
	case "err_unsupported":
		return api.ErrUnsupported
	case "err_no_service":
		return api.ErrNoService
	case "err_closed":
		return api.ErrClosed
	case "err_unknown_transaction":
		return api.ErrUnknownTransaction
	default:
		return errors.New(e.Message)
	}
}
