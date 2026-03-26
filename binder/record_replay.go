package binder

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type RecordedError struct {
	Kind          string
	Message       string
	StatusCode    int32
	ExceptionCode ExceptionCode
}

type TransactionRecord struct {
	Descriptor     string
	Code           uint32
	Flags          Flags
	RequestBytes   []byte
	RequestObjects []ParcelObject
	ReplyBytes     []byte
	ReplyObjects   []ParcelObject
	Error          *RecordedError
}

type TransactionRecorder struct {
	mu         sync.Mutex
	descriptor string
	records    []TransactionRecord
}

func NewTransactionRecorder() *TransactionRecorder {
	return &TransactionRecorder{}
}

func (r *TransactionRecorder) Descriptor() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.descriptor
}

func (r *TransactionRecorder) Records() []TransactionRecord {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]TransactionRecord, 0, len(r.records))
	for _, record := range r.records {
		out = append(out, cloneTransactionRecord(record))
	}
	return out
}

func NewRecordingBinder(target Binder, recorder *TransactionRecorder) Binder {
	if recorder == nil {
		recorder = NewTransactionRecorder()
	}
	return &recordingBinder{
		target:   target,
		recorder: recorder,
	}
}

func NewReplayBinder(records []TransactionRecord) Binder {
	copied := make([]TransactionRecord, 0, len(records))
	descriptor := ""
	for _, record := range records {
		cloned := cloneTransactionRecord(record)
		copied = append(copied, cloned)
		if descriptor == "" && cloned.Descriptor != "" {
			descriptor = cloned.Descriptor
		}
	}
	if descriptor == "" {
		descriptor = "replay"
	}
	return &replayBinder{
		descriptor: descriptor,
		records:    copied,
	}
}

type recordingBinder struct {
	target   Binder
	recorder *TransactionRecorder
}

func (b *recordingBinder) Descriptor(ctx context.Context) (string, error) {
	if b == nil || b.target == nil {
		return "", ErrUnsupported
	}
	desc, err := b.target.Descriptor(ctx)
	if err == nil && b.recorder != nil {
		b.recorder.mu.Lock()
		if b.recorder.descriptor == "" {
			b.recorder.descriptor = desc
		}
		b.recorder.mu.Unlock()
	}
	return desc, err
}

func (b *recordingBinder) Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error) {
	if b == nil || b.target == nil {
		return nil, ErrUnsupported
	}

	requestBytes, requestObjects := cloneParcel(data)
	reply, err := b.target.Transact(ctx, code, data, flags)
	replyBytes, replyObjects := cloneParcel(reply)
	descriptor, _ := b.Descriptor(context.Background())

	if b.recorder != nil {
		b.recorder.mu.Lock()
		if b.recorder.descriptor == "" {
			b.recorder.descriptor = descriptor
		}
		b.recorder.records = append(b.recorder.records, TransactionRecord{
			Descriptor:     descriptor,
			Code:           code,
			Flags:          flags,
			RequestBytes:   requestBytes,
			RequestObjects: requestObjects,
			ReplyBytes:     replyBytes,
			ReplyObjects:   replyObjects,
			Error:          captureRecordedError(err),
		})
		b.recorder.mu.Unlock()
	}
	return reply, err
}

func (b *recordingBinder) WatchDeath(ctx context.Context) (Subscription, error) {
	if b == nil || b.target == nil {
		return nil, ErrUnsupported
	}
	return b.target.WatchDeath(ctx)
}

func (b *recordingBinder) Close() error {
	if b == nil || b.target == nil {
		return nil
	}
	return b.target.Close()
}

type replayBinder struct {
	mu         sync.Mutex
	descriptor string
	records    []TransactionRecord
	index      int
}

func (b *replayBinder) Descriptor(ctx context.Context) (string, error) {
	if b == nil {
		return "", ErrUnsupported
	}
	return b.descriptor, nil
}

func (b *replayBinder) Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error) {
	if b == nil {
		return nil, ErrUnsupported
	}

	requestBytes, requestObjects := cloneParcel(data)

	b.mu.Lock()
	if b.index >= len(b.records) {
		b.mu.Unlock()
		return nil, ErrUnsupported
	}
	record := cloneTransactionRecord(b.records[b.index])
	b.index++
	b.mu.Unlock()

	if record.Code != code || record.Flags != flags || !equalParcelBytes(record.RequestBytes, requestBytes) || !equalParcelObjects(record.RequestObjects, requestObjects) {
		return nil, fmt.Errorf("%w: replay mismatch for %s code=%d", ErrBadParcelable, b.descriptor, code)
	}
	if record.Error != nil {
		return nil, record.Error.replay()
	}
	if record.ReplyBytes == nil && record.ReplyObjects == nil {
		return nil, nil
	}
	return NewParcelWire(record.ReplyBytes, record.ReplyObjects), nil
}

func (b *replayBinder) WatchDeath(ctx context.Context) (Subscription, error) {
	return nil, ErrUnsupported
}

func (b *replayBinder) Close() error {
	return nil
}

func captureRecordedError(err error) *RecordedError {
	if err == nil {
		return nil
	}

	var statusErr *StatusCodeError
	if errors.As(err, &statusErr) {
		return &RecordedError{
			Kind:       "status_code",
			StatusCode: statusErr.Code,
		}
	}

	var remoteErr *RemoteException
	if errors.As(err, &remoteErr) {
		return &RecordedError{
			Kind:          "remote_exception",
			Message:       remoteErr.Message,
			ExceptionCode: remoteErr.Code,
		}
	}

	switch {
	case errors.Is(err, ErrDeadObject):
		return &RecordedError{Kind: "err_dead_object"}
	case errors.Is(err, ErrFailedTxn):
		return &RecordedError{Kind: "err_failed_txn"}
	case errors.Is(err, ErrBadParcelable):
		return &RecordedError{Kind: "err_bad_parcelable"}
	case errors.Is(err, ErrPermissionDenied):
		return &RecordedError{Kind: "err_permission_denied"}
	case errors.Is(err, ErrUnsupported):
		return &RecordedError{Kind: "err_unsupported"}
	case errors.Is(err, ErrNoService):
		return &RecordedError{Kind: "err_no_service"}
	case errors.Is(err, ErrClosed):
		return &RecordedError{Kind: "err_closed"}
	case errors.Is(err, ErrUnknownTransaction):
		return &RecordedError{Kind: "err_unknown_transaction"}
	default:
		return &RecordedError{
			Kind:    "text",
			Message: err.Error(),
		}
	}
}

func (e *RecordedError) replay() error {
	if e == nil {
		return nil
	}

	switch e.Kind {
	case "status_code":
		return &StatusCodeError{Code: e.StatusCode}
	case "remote_exception":
		return &RemoteException{Code: e.ExceptionCode, Message: e.Message}
	case "err_dead_object":
		return ErrDeadObject
	case "err_failed_txn":
		return ErrFailedTxn
	case "err_bad_parcelable":
		return ErrBadParcelable
	case "err_permission_denied":
		return ErrPermissionDenied
	case "err_unsupported":
		return ErrUnsupported
	case "err_no_service":
		return ErrNoService
	case "err_closed":
		return ErrClosed
	case "err_unknown_transaction":
		return ErrUnknownTransaction
	default:
		return errors.New(e.Message)
	}
}

func cloneTransactionRecord(record TransactionRecord) TransactionRecord {
	return TransactionRecord{
		Descriptor:     record.Descriptor,
		Code:           record.Code,
		Flags:          record.Flags,
		RequestBytes:   append([]byte(nil), record.RequestBytes...),
		RequestObjects: cloneParcelObjects(record.RequestObjects),
		ReplyBytes:     append([]byte(nil), record.ReplyBytes...),
		ReplyObjects:   cloneParcelObjects(record.ReplyObjects),
		Error:          cloneRecordedError(record.Error),
	}
}

func cloneRecordedError(err *RecordedError) *RecordedError {
	if err == nil {
		return nil
	}
	clone := *err
	return &clone
}

func cloneParcel(parcel *Parcel) ([]byte, []ParcelObject) {
	if parcel == nil {
		return nil, nil
	}
	return parcel.Bytes(), parcel.Objects()
}

func cloneParcelObjects(objects []ParcelObject) []ParcelObject {
	if len(objects) == 0 {
		return nil
	}
	out := make([]ParcelObject, len(objects))
	copy(out, objects)
	return out
}

func equalParcelBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalParcelObjects(a, b []ParcelObject) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
