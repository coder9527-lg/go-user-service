package controller_test

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// 每个测试用例分配独立命名的共享内存库，保证并发连接可见同一份数据，
// 同时不同用例之间互不干扰（替代每连接独立的 ":memory:"）。
var testDBSeq uint64

func sharedMemoryDSN(t *testing.T) string {
	t.Helper()
	n := atomic.AddUint64(&testDBSeq, 1)
	return fmt.Sprintf("file:webtestdb%d?mode=memory&cache=shared", n)
}
