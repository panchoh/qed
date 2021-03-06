/*
   Copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, S.A.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package raftwal

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/bbva/qed/protocol"

	"github.com/bbva/qed/log"
	"github.com/bbva/qed/storage/rocks"
	metrics_utils "github.com/bbva/qed/testutils/metrics"
	utilrand "github.com/bbva/qed/testutils/rand"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

func init() {
	log.SetLogger("testRaft", log.DEBUG)

}

func raftAddr(id int) string {
	return fmt.Sprintf(":1830%d", id)
}

func newNode(t *testing.T, id int) (*RaftBalloon, func()) {
	dbPath := fmt.Sprintf("/var/tmp/raft-test/node%d/db", id)

	err := os.MkdirAll(dbPath, os.FileMode(0755))
	require.NoError(t, err)
	db, err := rocks.NewRocksDBStore(dbPath)
	require.NoError(t, err)

	raftPath := fmt.Sprintf("/var/tmp/raft-test/node%d/raft", id)
	err = os.MkdirAll(raftPath, os.FileMode(0755))
	require.NoError(t, err)
	r, err := NewRaftBalloon(raftPath, raftAddr(id), fmt.Sprintf("%d", id), db, make(chan *protocol.Snapshot, 25000))
	require.NoError(t, err)

	return r, func() {
		os.RemoveAll(fmt.Sprintf("/var/tmp/raft-test/node%d", id))
	}
}

func Test_Raft_IsLeader(t *testing.T) {

	log.SetLogger("Test_Raft_IsLeader", log.SILENT)

	r, clean := newNode(t, 1)
	defer clean()

	err := r.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	defer func() {
		err = r.Close(true)
		require.NoError(t, err)
	}()

	_, err = r.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	require.True(t, r.IsLeader(), "single node is not leader!")

}

func Test_Raft_OpenStore_CloseSingleNode(t *testing.T) {

	r, clean := newNode(t, 2)
	defer clean()

	err := r.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	err = r.Close(true)
	require.NoError(t, err)

	err = r.Open(true, map[string]string{"foo": "bar"})
	require.Equal(t, err, ErrBalloonInvalidState, err, "incorrect error returned on re-open attempt")

}

func Test_Raft_MultiNode_Join(t *testing.T) {

	log.SetLogger("Test_Raft_MultiNodeJoin", log.SILENT)

	r0, clean0 := newNode(t, 3)
	defer func() {
		err := r0.Close(true)
		require.NoError(t, err)
		clean0()
	}()

	err := r0.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	r1, clean1 := newNode(t, 4)
	defer func() {
		err := r1.Close(true)
		require.NoError(t, err)
		clean1()
	}()

	err = r1.Open(false, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	err = r0.Join("1", string(r1.raft.transport.LocalAddr()), map[string]string{"foo": "bar"})
	require.NoError(t, err)

}

func Test_Raft_MultiNode_JoinRemove(t *testing.T) {

	r0, clean0 := newNode(t, 5)
	defer func() {
		err := r0.Close(true)
		require.NoError(t, err)
		clean0()
	}()

	err := r0.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	r1, clean1 := newNode(t, 6)
	defer func() {
		err := r1.Close(true)
		require.NoError(t, err)
		clean1()
	}()

	err = r1.Open(false, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	err = r0.Join("6", string(r1.raft.transport.LocalAddr()), map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	// Check leader state on follower.
	require.Equal(t, r1.LeaderAddr(), r0.Addr(), "wrong leader address returned")

	id, err := r1.LeaderID()
	require.NoError(t, err)

	require.Equal(t, id, r0.ID(), "wrong leader ID returned")

	storeNodes := []string{r0.id, r1.id}
	sort.StringSlice(storeNodes).Sort()

	nodes, err := r0.Nodes()
	require.NoError(t, err)
	require.Equal(t, len(nodes), len(storeNodes), "size of cluster is not correct")

	if storeNodes[0] != string(nodes[0].ID) || storeNodes[1] != string(nodes[1].ID) {
		t.Fatalf("cluster does not have correct nodes")
	}

	// Remove a node.
	err = r0.Remove(r1.ID())
	require.NoError(t, err)

	nodes, err = r0.Nodes()
	require.NoError(t, err)

	require.Equal(t, len(nodes), 1, "size of cluster is not correct post remove")
	require.Equal(t, r0.ID(), string(nodes[0].ID), "cluster does not have correct nodes post remove")

}

func Test_Raft_SingleNode_SnapshotOnDisk(t *testing.T) {
	r0, clean0 := newNode(t, 7)

	err := r0.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	// Add event
	rand.Seed(42)
	expectedBalloonVersion := uint64(rand.Intn(50))
	for i := uint64(0); i < expectedBalloonVersion; i++ {
		_, err = r0.Add([]byte(fmt.Sprintf("Test Event %d", i)))
		require.NoError(t, err)
	}
	// force snapshot
	// Snap the node and write to disk.
	snapshot, err := r0.fsm.Snapshot()
	require.NoError(t, err)

	snapDir := mustTempDir()
	defer os.RemoveAll(snapDir)
	snapFile, err := os.Create(filepath.Join(snapDir, "snapshot"))
	require.NoError(t, err)

	sink := &mockSnapshotSink{snapFile}
	err = snapshot.Persist(sink)
	require.NoError(t, err)

	// Check restoration.
	snapFile, err = os.Open(filepath.Join(snapDir, "snapshot"))
	require.NoError(t, err)

	err = r0.Close(true)
	require.NoError(t, err)
	clean0()

	r8, clean8 := newNode(t, 8)
	defer func() {
		err = r8.Close(true)
		require.NoError(t, err)
		clean8()
	}()

	err = r8.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r8.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	err = r8.fsm.Restore(snapFile)
	require.NoError(t, err)

	require.Equal(t, expectedBalloonVersion, r8.fsm.balloon.Version(), "Error in state recovery from snapshot")

}

func Test_Raft_SingleNode_SnapshotConsistency(t *testing.T) {
	r0, clean0 := newNode(t, 8)

	err := r0.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	// Add event
	rand.Seed(42)
	expectedBalloonVersion := uint64(9000)

	var wg, wgSnap sync.WaitGroup
	var snapshot raft.FSMSnapshot

	wg.Add(1)
	wgSnap.Add(1)
	go func() {
		defer wg.Done()
		for i := uint64(0); i < 20000; i++ {
			if i == expectedBalloonVersion {
				// force snapshot
				// Snap the node and write to disk.
				snapshot, err = r0.fsm.Snapshot()
				require.NoError(t, err)
				wgSnap.Done()
			}
			_, err = r0.Add([]byte(fmt.Sprintf("Test Event %d", i)))
			require.NoError(t, err)
		}
	}()

	wgSnap.Wait()

	snapDir := mustTempDir()
	// defer os.RemoveAll(snapDir)
	snapFile, err := os.Create(filepath.Join(snapDir, "snapshot"))
	require.NoError(t, err)

	sink := &mockSnapshotSink{snapFile}
	err = snapshot.Persist(sink)
	require.NoError(t, err)

	// Check restoration.
	snapFile, err = os.Open(filepath.Join(snapDir, "snapshot"))
	require.NoError(t, err)

	wg.Wait()
	err = r0.Close(true)
	require.NoError(t, err)
	clean0()

	r9, clean9 := newNode(t, 9)
	defer func() {
		err = r9.Close(true)
		require.NoError(t, err)
		clean9()
	}()

	err = r9.Open(true, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	_, err = r9.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	err = r9.fsm.Restore(snapFile)
	require.NoError(t, err)

	require.Equal(t, expectedBalloonVersion, r9.fsm.balloon.Version(), "Error in state recovery from snapshot")

}

func Test_Raft_MultiNode_WithMetadata(t *testing.T) {

	log.SetLogger("Test_Raft_MultiNodeMetadata", log.SILENT)

	r0, clean0 := newNode(t, 0)
	defer func() {
		err := r0.Close(true)
		require.NoError(t, err)
		clean0()
	}()

	err := r0.Open(true, map[string]string{"nodeID": "0"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	r1, clean1 := newNode(t, 1)
	defer func() {
		err := r1.Close(true)
		require.NoError(t, err)
		clean1()
	}()

	empty_meta := map[string]string{}
	err = r1.Open(false, empty_meta)
	require.NoError(t, err)

	err = r0.Join("1", string(r1.raft.transport.LocalAddr()), map[string]string{"nodeID": "1"})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	require.Equal(t, r0.Info()["meta"], r1.Info()["meta"], "Both nodes must have the same metadata.")
}

func Test_Raft_MultiNode_Remove_WithMetadata(t *testing.T) {

	log.SetLogger("Test_Raft_MultiNodeMetadataRemove", log.SILENT)

	// Node 0
	r0, clean0 := newNode(t, 0)
	defer func() {
		err := r0.Close(true)
		require.NoError(t, err)
		clean0()
	}()

	err := r0.Open(true, map[string]string{"nodeID": "0"})
	require.NoError(t, err)

	_, err = r0.WaitForLeader(10 * time.Second)
	require.NoError(t, err)

	// Node 1
	r1, clean1 := newNode(t, 1)

	empty_meta := map[string]string{}
	err = r1.Open(false, empty_meta)
	require.NoError(t, err)

	// Node 2
	r2, clean2 := newNode(t, 2)
	defer func() {
		err := r2.Close(true)
		require.NoError(t, err)
		clean2()
	}()

	err = r2.Open(false, empty_meta)
	require.NoError(t, err)

	// Join
	err = r0.Join("1", string(r1.raft.transport.LocalAddr()), map[string]string{"nodeID": "1"})
	require.NoError(t, err)
	err = r0.Join("2", string(r2.raft.transport.LocalAddr()), map[string]string{"nodeID": "2"})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// Test
	require.Equal(t, len(r0.Info()["meta"].(map[string]map[string]string)), 3, "Node 0 metadata should have info of 3 nodes.")

	// Kill & remove Node 1
	err = r1.Close(true)
	require.NoError(t, err)
	clean1()

	err = r0.Remove(r1.ID())
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Test
	require.Equal(t, r0.Info()["meta"], r2.Info()["meta"], "All nodes must have the same metadata.")
	require.Equal(t, len(r0.Info()["meta"].(map[string]map[string]string)), 2, "Node 0 metadata should have info of 2 nodes.")
}

type mockSnapshotSink struct {
	*os.File
}

func (m *mockSnapshotSink) ID() string {
	return "1"
}

func (m *mockSnapshotSink) Cancel() error {
	return nil
}

func mustTempDir() string {
	var err error
	path, err := ioutil.TempDir("", "raft-test-")
	if err != nil {
		panic("failed to create temp dir")
	}
	return path
}

func newNodeBench(b *testing.B, id int) (*RaftBalloon, func()) {
	storePath := fmt.Sprintf("/var/tmp/raft-test/node%d/db", id)

	err := os.MkdirAll(storePath, os.FileMode(0755))
	require.NoError(b, err)
	store, err := rocks.NewRocksDBStore(storePath)
	require.NoError(b, err)

	raftPath := fmt.Sprintf("/var/tmp/raft-test/node%d/raft", id)
	err = os.MkdirAll(raftPath, os.FileMode(0755))
	require.NoError(b, err)

	snapshotsCh := make(chan *protocol.Snapshot, 10000)
	snapshotsDrainer(snapshotsCh)

	node, err := NewRaftBalloon(raftPath, raftAddr(id), fmt.Sprintf("%d", id), store, snapshotsCh)
	require.NoError(b, err)

	srvCloseF := metrics_utils.StartMetricsServer(node, store)

	return node, func() {
		srvCloseF()
		close(snapshotsCh)
		os.RemoveAll(fmt.Sprintf("/var/tmp/raft-test/node%d", id))
	}

}

func snapshotsDrainer(snapshotsCh chan *protocol.Snapshot) {
	go func() {
		for {
			_, ok := <-snapshotsCh
			if !ok {
				return
			}
		}
	}()
}

func BenchmarkRaftAdd(b *testing.B) {

	log.SetLogger("BenchmarkRaftAdd", log.SILENT)

	raftNode, clean := newNodeBench(b, 1)
	defer clean()

	err := raftNode.Open(true, map[string]string{"foo": "bar"})
	require.NoError(b, err)

	// b.N shoul be eq or greater than 500k to avoid benchmark framework spreading more than one goroutine.
	b.N = 2000000
	b.ResetTimer()
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := utilrand.Bytes(128)
			_, err := raftNode.Add(event)
			require.NoError(b, err)
		}
	})

}

func BenchmarkRaftAddBulk(b *testing.B) {

	log.SetLogger("BenchmarkRaftAddBulk", log.SILENT)

	raftNode, clean := newNodeBench(b, 1)
	defer clean()

	err := raftNode.Open(true, map[string]string{"foo": "bar"})
	require.NoError(b, err)

	// b.N shoul be eq or greater than 500k to avoid benchmark framework spreading more than one goroutine.
	b.N = 2000000
	b.ResetTimer()
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			events := [][]byte{utilrand.Bytes(128)}
			_, err := raftNode.AddBulk(events)
			require.NoError(b, err)
		}
	})

}
