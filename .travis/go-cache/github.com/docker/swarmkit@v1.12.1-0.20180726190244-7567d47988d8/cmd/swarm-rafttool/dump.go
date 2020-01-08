package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/snap"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/docker/swarmkit/manager/state/raft/storage"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/gogo/protobuf/proto"
)

func loadData(swarmdir, unlockKey string) (*storage.WALData, *raftpb.Snapshot, error) {
	snapDir := filepath.Join(swarmdir, "raft", "snap-v3-encrypted")
	walDir := filepath.Join(swarmdir, "raft", "wal-v3-encrypted")

	var (
		snapFactory storage.SnapFactory
		walFactory  storage.WALFactory
	)

	_, err := os.Stat(walDir)
	if err == nil {
		// Encrypted WAL is present
		krw, err := getKRW(swarmdir, unlockKey)
		if err != nil {
			return nil, nil, err
		}
		deks, err := getDEKData(krw)
		if err != nil {
			return nil, nil, err
		}

		// always set FIPS=false, because we want to decrypt logs stored using any
		// algorithm, not just FIPS-compatible ones
		_, d := encryption.Defaults(deks.CurrentDEK, false)
		if deks.PendingDEK == nil {
			_, d2 := encryption.Defaults(deks.PendingDEK, false)
			d = encryption.NewMultiDecrypter(d, d2)
		}

		walFactory = storage.NewWALFactory(encryption.NoopCrypter, d)
		snapFactory = storage.NewSnapFactory(encryption.NoopCrypter, d)
	} else {
		// Try unencrypted WAL
		snapDir = filepath.Join(swarmdir, "raft", "snap")
		walDir = filepath.Join(swarmdir, "raft", "wal")

		walFactory = storage.OriginalWAL
		snapFactory = storage.OriginalSnap
	}

	var walsnap walpb.Snapshot
	snapshot, err := snapFactory.New(snapDir).Load()
	if err != nil && err != snap.ErrNoSnapshot {
		return nil, nil, err
	}
	if snapshot != nil {
		walsnap.Index = snapshot.Metadata.Index
		walsnap.Term = snapshot.Metadata.Term
	}

	wal, walData, err := storage.ReadRepairWAL(context.Background(), walDir, walsnap, walFactory)
	if err != nil {
		return nil, nil, err
	}
	wal.Close()

	return &walData, snapshot, nil
}

func dumpWAL(swarmdir, unlockKey string, start, end uint64, redact bool) error {
	walData, _, err := loadData(swarmdir, unlockKey)
	if err != nil {
		return err
	}

	for _, ent := range walData.Entries {
		if (start == 0 || ent.Index >= start) && (end == 0 || ent.Index <= end) {
			fmt.Printf("Entry Index=%d, Term=%d, Type=%s:\n", ent.Index, ent.Term, ent.Type.String())
			switch ent.Type {
			case raftpb.EntryConfChange:
				cc := &raftpb.ConfChange{}
				err := proto.Unmarshal(ent.Data, cc)
				if err != nil {
					return err
				}

				fmt.Println("Conf change type:", cc.Type.String())
				fmt.Printf("Node ID: %x\n\n", cc.NodeID)

			case raftpb.EntryNormal:
				r := &api.InternalRaftRequest{}
				err := proto.Unmarshal(ent.Data, r)
				if err != nil {
					return err
				}

				if redact {
					// redact sensitive information
					for _, act := range r.Action {
						target := act.GetTarget()
						switch actype := target.(type) {
						case *api.StoreAction_Cluster:
							actype.Cluster.UnlockKeys = []*api.EncryptionKey{}
							actype.Cluster.NetworkBootstrapKeys = []*api.EncryptionKey{}
							actype.Cluster.RootCA = api.RootCA{}
							actype.Cluster.Spec.CAConfig = api.CAConfig{}
						case *api.StoreAction_Secret:
							actype.Secret.Spec.Data = []byte("SECRET REDACTED")
						case *api.StoreAction_Config:
							actype.Config.Spec.Data = []byte("CONFIG REDACTED")
						case *api.StoreAction_Task:
							if container := actype.Task.Spec.GetContainer(); container != nil {
								container.Env = []string{"ENVVARS REDACTED"}
								if container.PullOptions != nil {
									container.PullOptions.RegistryAuth = "REDACTED"
								}
							}
						case *api.StoreAction_Service:
							if container := actype.Service.Spec.Task.GetContainer(); container != nil {
								container.Env = []string{"ENVVARS REDACTED"}
								if container.PullOptions != nil {
									container.PullOptions.RegistryAuth = "REDACTED"
								}
							}
						}
					}
				}

				if err := proto.MarshalText(os.Stdout, r); err != nil {
					return err
				}
				fmt.Println()
			}
		}
	}

	return nil
}

func dumpSnapshot(swarmdir, unlockKey string, redact bool) error {
	_, snapshot, err := loadData(swarmdir, unlockKey)
	if err != nil {
		return err
	}

	if snapshot == nil {
		return errors.New("no snapshot found")
	}

	s := &api.Snapshot{}
	if err := proto.Unmarshal(snapshot.Data, s); err != nil {
		return err
	}
	if s.Version != api.Snapshot_V0 {
		return fmt.Errorf("unrecognized snapshot version %d", s.Version)
	}

	fmt.Println("Active members:")
	for _, member := range s.Membership.Members {
		fmt.Printf(" NodeID=%s, RaftID=%x, Addr=%s\n", member.NodeID, member.RaftID, member.Addr)
	}
	fmt.Println()

	fmt.Println("Removed members:")
	for _, member := range s.Membership.Removed {
		fmt.Printf(" RaftID=%x\n", member)
	}
	fmt.Println()

	if redact {
		for _, cluster := range s.Store.Clusters {
			if cluster != nil {
				// expunge everything that may have key material
				cluster.RootCA = api.RootCA{}
				cluster.NetworkBootstrapKeys = []*api.EncryptionKey{}
				cluster.UnlockKeys = []*api.EncryptionKey{}
				cluster.Spec.CAConfig = api.CAConfig{}
			}
		}
		for _, secret := range s.Store.Secrets {
			if secret != nil {
				secret.Spec.Data = []byte("SECRET REDACTED")
			}
		}
		for _, config := range s.Store.Configs {
			if config != nil {
				config.Spec.Data = []byte("CONFIG REDACTED")
			}
		}
		for _, task := range s.Store.Tasks {
			if task != nil {
				if container := task.Spec.GetContainer(); container != nil {
					container.Env = []string{"ENVVARS REDACTED"}
					container.PullOptions.RegistryAuth = "REDACTED"
				}
			}
		}
		for _, service := range s.Store.Services {
			if service != nil {
				if container := service.Spec.Task.GetContainer(); container != nil {
					container.Env = []string{"ENVVARS REDACTED"}
					container.PullOptions.RegistryAuth = "REDACTED"
				}
			}
		}
	}

	fmt.Println("Objects:")
	if err := proto.MarshalText(os.Stdout, &s.Store); err != nil {
		return err
	}
	fmt.Println()

	return nil
}

// objSelector provides some criteria to select objects.
type objSelector struct {
	all  bool
	id   string
	name string
}

func bySelection(selector objSelector) store.By {
	if selector.all {
		return store.All
	}
	if selector.name != "" {
		return store.ByName(selector.name)
	}

	// find nothing
	return store.Or()
}

func dumpObject(swarmdir, unlockKey, objType string, selector objSelector) error {
	memStore := store.NewMemoryStore(nil)
	defer memStore.Close()

	walData, snapshot, err := loadData(swarmdir, unlockKey)
	if err != nil {
		return err
	}

	if snapshot != nil {
		var s api.Snapshot
		if err := s.Unmarshal(snapshot.Data); err != nil {
			return err
		}
		if s.Version != api.Snapshot_V0 {
			return fmt.Errorf("unrecognized snapshot version %d", s.Version)
		}

		if err := memStore.Restore(&s.Store); err != nil {
			return err
		}
	}

	for _, ent := range walData.Entries {
		if snapshot != nil && ent.Index <= snapshot.Metadata.Index {
			continue
		}

		if ent.Type != raftpb.EntryNormal {
			continue
		}

		r := &api.InternalRaftRequest{}
		err := proto.Unmarshal(ent.Data, r)
		if err != nil {
			return err
		}

		if r.Action != nil {
			if err := memStore.ApplyStoreActions(r.Action); err != nil {
				return err
			}
		}
	}

	var objects []proto.Message
	memStore.View(func(tx store.ReadTx) {
		switch objType {
		case "node":
			if selector.id != "" {
				object := store.GetNode(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Node
			results, err = store.FindNodes(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "service":
			if selector.id != "" {
				object := store.GetService(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Service
			results, err = store.FindServices(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "task":
			if selector.id != "" {
				object := store.GetTask(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Task
			results, err = store.FindTasks(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "network":
			if selector.id != "" {
				object := store.GetNetwork(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Network
			results, err = store.FindNetworks(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "cluster":
			if selector.id != "" {
				object := store.GetCluster(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Cluster
			results, err = store.FindClusters(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "secret":
			if selector.id != "" {
				object := store.GetSecret(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Secret
			results, err = store.FindSecrets(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "config":
			if selector.id != "" {
				object := store.GetConfig(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Config
			results, err = store.FindConfigs(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "resource":
			if selector.id != "" {
				object := store.GetResource(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Resource
			results, err = store.FindResources(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		case "extension":
			if selector.id != "" {
				object := store.GetExtension(tx, selector.id)
				if object != nil {
					objects = append(objects, object)
				}
			}

			var results []*api.Extension
			results, err = store.FindExtensions(tx, bySelection(selector))
			if err != nil {
				return
			}
			for _, o := range results {
				objects = append(objects, o)
			}
		default:
			err = fmt.Errorf("unrecognized object type %s", objType)
		}
	})

	if err != nil {
		return err
	}

	if len(objects) == 0 {
		return fmt.Errorf("no matching objects found")
	}

	for _, object := range objects {
		if err := proto.MarshalText(os.Stdout, object); err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}
