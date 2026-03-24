package protocol

import api "libbinder-go/binder"

// Status models the internal split between transport failure and remote exception.
type Status struct {
	TransportErr error
	Remote       *RemoteException
}

func (s Status) IsOK() bool {
	return s.TransportErr == nil && s.Remote == nil
}

// RemoteException is the backend-facing form of an exception returned by a remote Binder peer.
type RemoteException struct {
	Code        api.ExceptionCode
	Message     string
	ServiceCode int32
}
