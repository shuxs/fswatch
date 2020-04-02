package fswatch

import "bytes"

type Event struct {
	Name string
	Op   Op
}

type Op uint32

const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

func (op Op) String() string {
	var buffer bytes.Buffer
	if op&Create == Create {
		buffer.WriteString("|创建")
	}
	if op&Remove == Remove {
		buffer.WriteString("|删除")
	}
	if op&Write == Write {
		buffer.WriteString("|修改")
	}
	if op&Rename == Rename {
		buffer.WriteString("|改名")
	}
	if op&Chmod == Chmod {
		buffer.WriteString("|属性")
	}
	if buffer.Len() == 0 {
		return ""
	}
	return buffer.String()[1:] // Strip leading pipe
}

func (op Op) Is(anotherOps ...Op) bool {
	for _, anotherOp := range anotherOps {
		if op&anotherOp == anotherOp {
			return true
		}
	}
	return false
}
