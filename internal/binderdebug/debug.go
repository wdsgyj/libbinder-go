package binderdebug

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type PIDInfo struct {
	RefPIDs     map[uint64][]int
	ThreadUsage uint32
	ThreadCount uint32
}

type Reader struct {
	contextName      string
	procRoots        []string
	transactionPaths []string
}

func NewReader(driverPath string) Reader {
	return Reader{
		contextName: contextFromDriverPath(driverPath),
		procRoots: []string{
			"/dev/binderfs/binder_logs/proc",
			"/d/binder/proc",
		},
		transactionPaths: []string{
			"/dev/binderfs/binder_logs/transactions",
			"/d/binder/transactions",
		},
	}
}

func NewReaderWithPaths(driverPath string, procRoots []string, transactionPaths []string) Reader {
	r := NewReader(driverPath)
	if len(procRoots) != 0 {
		r.procRoots = append([]string(nil), procRoots...)
	}
	if len(transactionPaths) != 0 {
		r.transactionPaths = append([]string(nil), transactionPaths...)
	}
	return r
}

func (r Reader) GetPIDInfo(pid int) (PIDInfo, error) {
	info := PIDInfo{RefPIDs: map[uint64][]int{}}
	err := r.scanBinderContext(pid, func(line string) error {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "node "):
			ptr, pids, err := parseNodeReferenceLine(trimmed)
			if err != nil {
				return err
			}
			if ptr != 0 && len(pids) != 0 {
				info.RefPIDs[ptr] = append(info.RefPIDs[ptr], pids...)
			}
		case strings.HasPrefix(trimmed, "thread "):
			inUse, binderThread, err := parseThreadState(trimmed)
			if err != nil {
				return err
			}
			if !binderThread {
				return nil
			}
			if inUse {
				info.ThreadUsage++
			}
			info.ThreadCount++
		}
		return nil
	})
	return info, err
}

func (r Reader) GetClientPIDs(callerPID int, servicePID int, handle uint32) ([]int, error) {
	var node int
	err := r.scanBinderContext(callerPID, func(line string) error {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "ref ") {
			return nil
		}
		desc, parsedNode, err := parseRefLine(trimmed)
		if err != nil {
			return err
		}
		if desc == int(handle) {
			node = parsedNode
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var pids []int
	err = r.scanBinderContext(servicePID, func(line string) error {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "node ") {
			return nil
		}
		matchedNode, foundPIDs, err := parseNodePIDs(trimmed)
		if err != nil {
			return err
		}
		if matchedNode == node {
			pids = append(pids, foundPIDs...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Ints(pids)
	return pids, nil
}

func (r Reader) GetTransactions(pid int) (string, error) {
	file, err := openFirst(r.transactionPaths)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	var b strings.Builder
	prefix := "proc " + strconv.Itoa(pid)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			b.WriteString(line)
			b.WriteByte('\n')
			for scanner.Scan() {
				line = scanner.Text()
				if strings.HasPrefix(line, "proc ") {
					return b.String(), nil
				}
				b.WriteString(line)
				b.WriteByte('\n')
			}
			return b.String(), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}

func (r Reader) scanBinderContext(pid int, eachLine func(string) error) error {
	name := strconv.Itoa(pid)
	paths := make([]string, 0, len(r.procRoots))
	for _, root := range r.procRoots {
		paths = append(paths, filepath.Join(root, name))
	}
	file, err := openFirst(paths)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	isDesiredContext := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "context") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				isDesiredContext = false
				continue
			}
			isDesiredContext = fields[len(fields)-1] == r.contextName
			continue
		}
		if !isDesiredContext {
			continue
		}
		if err := eachLine(line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func openFirst(paths []string) (*os.File, error) {
	var lastErr error
	for _, path := range paths {
		file, err := os.Open(path)
		if err == nil {
			return file, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, os.ErrNotExist
}

func contextFromDriverPath(driverPath string) string {
	name := filepath.Base(driverPath)
	switch name {
	case "", ".", string(filepath.Separator):
		return "binder"
	case "binder", "hwbinder", "vndbinder":
		return name
	default:
		return name
	}
}

func parseNodeReferenceLine(line string) (uint64, []int, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, nil, nil
	}

	var ptr uint64
	var seenProc bool
	var pids []int
	for _, token := range fields {
		switch {
		case strings.HasPrefix(token, "u"):
			parsed, err := strconv.ParseUint("0x"+strings.TrimPrefix(token, "u"), 0, 64)
			if err != nil {
				return 0, nil, err
			}
			ptr = parsed
		case token == "proc":
			seenProc = true
		case seenProc:
			pid, err := strconv.Atoi(strings.TrimSuffix(token, ":"))
			if err != nil {
				return 0, nil, err
			}
			pids = append(pids, pid)
		}
	}
	return ptr, pids, nil
}

func parseThreadState(line string) (bool, bool, error) {
	pos := strings.Index(line, "l ")
	if pos == -1 || pos+3 >= len(line) {
		return false, false, fmt.Errorf("invalid thread line %q", line)
	}
	waitState := line[pos+2]
	threadState := line[pos+3]
	inUse := waitState != '1'
	binderThread := threadState != '0'
	return inUse, binderThread, nil
}

func parseRefLine(line string) (desc int, node int, err error) {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return 0, 0, fmt.Errorf("invalid ref line %q", line)
	}
	desc, err = strconv.Atoi(fields[3])
	if err != nil {
		return 0, 0, err
	}
	node, err = strconv.Atoi(fields[5])
	if err != nil {
		return 0, 0, err
	}
	return desc, node, nil
}

func parseNodePIDs(line string) (node int, pids []int, err error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, nil, fmt.Errorf("invalid node line %q", line)
	}
	node, err = strconv.Atoi(strings.TrimSuffix(fields[1], ":"))
	if err != nil {
		return 0, nil, err
	}
	seenProc := false
	for _, token := range fields[2:] {
		if token == "proc" {
			seenProc = true
			continue
		}
		if !seenProc {
			continue
		}
		pid, err := strconv.Atoi(token)
		if err != nil {
			return 0, nil, err
		}
		pids = append(pids, pid)
	}
	return node, pids, nil
}
