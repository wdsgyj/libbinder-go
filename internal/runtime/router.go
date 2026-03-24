package runtime

// TransactionRouter decides which backend and worker should execute a transaction.
type TransactionRouter struct{}

func NewTransactionRouter() *TransactionRouter {
	return &TransactionRouter{}
}
