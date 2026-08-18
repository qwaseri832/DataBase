package replication_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/filesystem"
	"github.com/qwaseri832/DataBase/internal/database/storage/replication"
	"github.com/qwaseri832/DataBase/internal/database/storage/wal"
	"github.com/qwaseri832/DataBase/internal/network"
)

func TestSlaveReceivesSegmentLargerThanBuffer(t *testing.T) {
	masterDir := t.TempDir()
	slaveDir := t.TempDir()
	logger := zap.NewNop()

	const records = 20_000
	want := make([]wal.Record, records)
	for i := range want {
		want[i] = wal.Record{
			LSN:  int64(i + 1),
			Op:   wal.OpSet,
			Args: []string{fmt.Sprintf("ключ-%05d", i), fmt.Sprintf("значение-%05d", i)},
		}
	}

	var payload bytes.Buffer
	for i := range want {
		if err := want[i].Encode(&payload); err != nil {
			t.Fatalf("подготовка сегмента: %v", err)
		}
	}
	if payload.Len() < 512<<10 {
		t.Fatalf("сегмент получился слишком мал для проверки: %d байт", payload.Len())
	}

	segName := filesystem.SegmentPrefix + "0000000000001_000001.log"
	if err := os.WriteFile(filepath.Join(masterDir, segName), payload.Bytes(), 0o644); err != nil {
		t.Fatalf("запись сегмента мастера: %v", err)
	}

	srv, err := network.ListenTCP("127.0.0.1:0", logger)
	if err != nil {
		t.Fatalf("ListenTCP: %v", err)
	}
	master := replication.NewMaster(srv, masterDir, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go master.Run(ctx)

	if !master.IsMaster() {
		t.Error("Master.IsMaster() = false")
	}

	client, err := network.DialTCP(srv.Addr().String(), network.WithClientTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("DialTCP: %v", err)
	}
	slave := replication.NewSlave(client, slaveDir, 10*time.Millisecond, logger)

	if slave.IsMaster() {
		t.Error("Slave.IsMaster() = true")
	}

	slaveCtx, stopSlave := context.WithCancel(context.Background())
	defer stopSlave()
	go slave.Run(slaveCtx)

	select {
	case got := <-slave.Stream():
		if len(got) != len(want) {
			t.Fatalf("получено записей: %d, ожидалось %d", len(got), len(want))
		}
		if !reflect.DeepEqual(got[0], want[0]) {
			t.Errorf("первая запись: получено %+v, ожидалось %+v", got[0], want[0])
		}
		if last := len(got) - 1; !reflect.DeepEqual(got[last], want[last]) {
			t.Errorf("последняя запись: получено %+v, ожидалось %+v", got[last], want[last])
		}
	case <-time.After(15 * time.Second):
		t.Fatal("реплика не получила сегмент")
	}

	if _, err := os.Stat(filepath.Join(slaveDir, segName)); err != nil {
		t.Errorf("сегмент не сохранён на реплике: %v", err)
	}

	stopSlave()
}

func TestMasterReportsNoNewSegments(t *testing.T) {
	dir := t.TempDir()
	logger := zap.NewNop()

	srv, err := network.ListenTCP("127.0.0.1:0", logger)
	if err != nil {
		t.Fatalf("ListenTCP: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go replication.NewMaster(srv, dir, logger).Run(ctx)

	client, err := network.DialTCP(srv.Addr().String(), network.WithClientTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("DialTCP: %v", err)
	}
	defer client.Close()

	req := replication.SyncRequest{}
	raw, err := replication.Encode(&req)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	respRaw, err := client.Send(raw)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	var resp replication.SyncResponse
	if err := replication.Decode(&resp, respRaw); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !resp.OK {
		t.Error("SyncResponse.OK = false")
	}
	if resp.SegmentName != "" {
		t.Errorf("SegmentName = %q, ожидалась пустая строка", resp.SegmentName)
	}
}
