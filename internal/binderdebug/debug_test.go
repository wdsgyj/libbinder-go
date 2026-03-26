package binderdebug

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetPIDInfo(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	if err := os.MkdirAll(procRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	const log = `
context binder
  node 66730: u00007590061890e0 c0000759036130950 pri 0:120 hs 1 hw 1 ls 0 lw 0 is 2 iw 2 tr 1 proc 2300 1790
  thread 2999: l 21 need_return 1 tr 0
  thread 3000: l 11 need_return 1 tr 0
context vndbinder
  thread 4000: l 00 need_return 1 tr 0
`
	if err := os.WriteFile(filepath.Join(procRoot, "100"), []byte(log), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	reader := NewReaderWithPaths("/dev/binder", []string{procRoot}, nil)
	info, err := reader.GetPIDInfo(100)
	if err != nil {
		t.Fatalf("GetPIDInfo: %v", err)
	}
	if info.ThreadCount != 2 || info.ThreadUsage != 1 {
		t.Fatalf("thread info = %#v, want count=2 usage=1", info)
	}
	ptr := uint64(0x00007590061890e0)
	if got := info.RefPIDs[ptr]; !reflect.DeepEqual(got, []int{2300, 1790}) {
		t.Fatalf("RefPIDs[%x] = %#v, want [2300 1790]", ptr, got)
	}
}

func TestGetClientPIDs(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	if err := os.MkdirAll(procRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	clientLog := `
context binder
  ref 52493: desc 910 node 52492 s 1 w 1 d 0000000000000000
`
	serviceLog := `
context binder
  node 52492: u00007803fc982e80 c000078042c982210 pri 0:139 hs 1 hw 1 ls 0 lw 0 is 2 iw 2 tr 1 proc 488 683
`
	if err := os.WriteFile(filepath.Join(procRoot, "100"), []byte(clientLog), 0o644); err != nil {
		t.Fatalf("WriteFile(client): %v", err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "200"), []byte(serviceLog), 0o644); err != nil {
		t.Fatalf("WriteFile(service): %v", err)
	}

	reader := NewReaderWithPaths("/dev/binder", []string{procRoot}, nil)
	pids, err := reader.GetClientPIDs(100, 200, 910)
	if err != nil {
		t.Fatalf("GetClientPIDs: %v", err)
	}
	if !reflect.DeepEqual(pids, []int{488, 683}) {
		t.Fatalf("GetClientPIDs = %#v, want [488 683]", pids)
	}
}

func TestGetPIDInfoUsesLastOpenError(t *testing.T) {
	root := t.TempDir()
	deniedRoot := filepath.Join(root, "denied", "proc")
	if err := os.MkdirAll(deniedRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Chmod(deniedRoot, 0o000); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(deniedRoot, 0o755)
	})

	reader := NewReaderWithPaths("/dev/binder", []string{deniedRoot, filepath.Join(root, "missing", "proc")}, nil)
	_, err := reader.GetPIDInfo(100)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("GetPIDInfo error = %v, want os.ErrNotExist", err)
	}
}
