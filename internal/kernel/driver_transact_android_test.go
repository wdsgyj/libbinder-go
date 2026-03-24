//go:build android

package kernel

import "testing"

func TestDriverProtocolVersionOnAndroid(t *testing.T) {
	driver := NewDriverManager(DefaultDriverPath)

	if err := driver.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := driver.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	version, err := driver.ProtocolVersion()
	if err != nil {
		t.Fatalf("ProtocolVersion: %v", err)
	}
	if version != BinderCurrentProtocolVersion {
		t.Fatalf("ProtocolVersion = %d, want %d", version, BinderCurrentProtocolVersion)
	}
}

func TestDriverWriteReadEmptyOnAndroid(t *testing.T) {
	driver := NewDriverManager(DefaultDriverPath)

	if err := driver.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := driver.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	var bwr BinderWriteRead
	if err := driver.WriteRead(&bwr); err != nil {
		t.Fatalf("WriteRead(empty): %v", err)
	}
	if bwr.WriteConsumed != 0 {
		t.Fatalf("WriteConsumed = %d, want 0", bwr.WriteConsumed)
	}
	if bwr.ReadConsumed != 0 {
		t.Fatalf("ReadConsumed = %d, want 0", bwr.ReadConsumed)
	}
}

func TestDriverPingContextManagerOnAndroid(t *testing.T) {
	driver := NewDriverManager(DefaultDriverPath)

	if err := driver.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := driver.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	if err := driver.SetMaxThreads(0); err != nil {
		t.Fatalf("SetMaxThreads: %v", err)
	}
	if err := driver.EnterLooper(); err != nil {
		t.Fatalf("EnterLooper: %v", err)
	}

	var tx BinderTransactionData
	tx.SetTargetHandle(0)
	tx.Code = PingTransaction

	reply, commands, err := driver.TransactHandle(0, PingTransaction, nil, 0)
	if err != nil {
		t.Fatalf("TransactHandle: %v", err)
	}

	if !containsCommand(commands, BRTransactionComplete) {
		t.Fatalf("commands = %v, want BR_TRANSACTION_COMPLETE", commandNames(commands))
	}
	if !containsCommand(commands, BRReply) {
		t.Fatalf("commands = %v, want BR_REPLY", commandNames(commands))
	}
	if len(reply) != 0 {
		t.Fatalf("len(reply) = %d, want 0", len(reply))
	}
}
