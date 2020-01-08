# Raft Encryption

The original high-level design document for raft encryption is https://docs.google.com/document/d/1YxMH2oIv-mtRcs1djRkm0ndLfiteo0UzBUFKYhLKYkQ/edit#heading=h.79rz783bo3q2.

The implementation-specific parts will be duplicated and elaborated on in this document, and this document will be kept up-to-date as the implementation changes.

## Terms/keys involved

- **Raft DEK (Data Encrypting Key)**
    - Usage: encrypt the WAL and snapshots when written to disk.
    - Unique per manager node
    - Stored: as a PEM header in the TLS key that is stored on disk
    - Generation: auto-generated 32 bytes of random data when manager first initialized, or on DEK rotation
    - Encryption: optionally encrypted with the manager unlock key
    - Rotation: when the cluster goes from non-autolocked->autolocked
    - Deleted: when a manager is demoted or leaves the cluster

- **TLS key**
    - Usage: authenticate a manager to the swarm cluster via mTLS
    - Unique per manager node
    - Stored: on disk in `swarm/certificates/swarm-node.key`
    - Generation: auto-generated Curve-P256 ECDSA key when node first joins swarm, or on certificate renewal
    - Encryption: optionally encrypted with the manager unlock key
    - Rotation:
        - when the cluster goes from non-autolocked->autolocked
        - when the TLS certificate is near expiry
        - when the manager changes role to worker or vice versa
        - when the cluster root CA changes
    - Deleted: when the node leaves the cluster

- **Manager unlock key**
    - Usage: encrypt the Raft DEK and TLS key (acts as a KEK, or key encrypting key)
    - Not unique per manager node - shared by all managers in the cluster
    - Stored: in raft (that's how it's propagated to all managers)
    - Generation: auto-generated 32 bytes of random data when the cluster is set to autolock, or when the unlock key us rotated
    - Encryption: like the rest of the raft store, via TLS in transit and via Raft DEK at rest
    - Rotation: via API
    - Deleted: when autolock is disabled


## Overview

The full raft store will be propagated to all managers encrypted only via mTLS (but not further encrypted in any way using the cluster unlock key).  This lets us store and propagate the unlock key using raft itself.  Any new manager that joins the cluster (authenticated via mTLS) will have access to all the data in cleartext, as well as the unlock key. A manager that falls behind after a key rotation will eventually get the latest unlock key and be able to encrypt using that unlock key.

When each node writes its periodic snapshots and WALs to disk, the write will go through an encryption layer.

Each manager generates a unique raft data encryption key (raft DEK).  The `etcd/wal` and `etcd/snap` packages are wrapped to encrypt using this key when saving entries and snapshots. Whenever a WAL entry or snapshot message is written to disk, the actual data will be encrypted.  The ciphertext, the algorithm, and IV serialized in a protobuf object (`MaybeEncryptedRecord`), and the WAL entry’s or snapshot message’s data field will be replaced with this serialized object.  The index, term, and other metadata in the WAL entry and snapshot message will remain unencrypted so that the underlying etcd/wal and etcd/snap packages can read the message and pass it up to the wrapping layer to decrypt.

The raft DEK, also serialized in a protobuf (`MaybeEncryptedRecord`) is written as a PEM header in the TLS key for the manager (which lives outside the raft store), so that the TLS key and the raft DEK can be re-encrypted and written in an atomic manner.

By default both the TLS key and raft DEK will be unencrypted, allowing a manager to restart from a stopped state without requiring any interaction from the user.  This mode should be considered the equivalent of completely unencrypted raft stores.  However, encrypting the raft data using a plaintext DEK allows us to simply remove or rotate the DEK in order to clear out the raft data, rather than have to carefully re-encrypt or remove all existing data.

The cluster can be configured to require all managers to auto-lock.  This means that a key encrypting key (KEK) will be generated, which will encrypt both the raft DEK (which encrypts raft logs on disk) and the TLS key (which lives outside of the raft store), since mTLS access to the rest of the raft cluster equates access to the entire unencrypted raft store.

## Encryption algorithms

By default, the WAL/snapshots, when written to disk, are encrypted using nacl/secretbox (in golang.org/x/crypto/nacl/secretbox), which uses XSalsa20 and Poly1305 to provide both encryption and authentication of small messages.  We generate random 24-byte nonces for each encrypted message.

When the raft DEK is encrypted, it is also encrypted using nacl/secretbox. The TLS key is encrypted with using the RFC 1423 implementation provided by golang.org/src/crypto/x509 using the AES-256-CBC PEM cipher.

If FIPS mode is enabled, the WAL/snapshots and the raft DEK are encrypted using fernet, which uses AES-128-CBC.  The TLS key is encrypted using PKCS#8 instead, which does not use md5 for a message digest.

## Raft DEK rotation

Raft DEK rotation is needed when going from a non-autolocked cluster to an autolocked cluster.  The DEK was previously available via plaintext, and could have been leaked, so we rotate the DEK.  This means we need to re-encrypt all the raft data using a new raft DEK  However, we do not want to take down all the managers in order to do this, or too severely impact their performance in order to re-encrypt all the raft data using the new DEK, we need to do the following:

1.  Start encrypting all new raft WALs and snapshots using the new key.  Assume that all new snapshots and WALs after index `i` will be encrypted using the new DEK.
1.  Trigger a snapshot that covers all indexes up to and including the index of the last WAL written using the old DEK: `0` to `i`.  This way, we no longer need the old WALs (and hence the old DEK) prior to `i` in order to bootstrap from disk - we can just load the snapshot (at index `i`, which will be encrypted using the new DEK.
1.  If there was already a snapshot for index `i`, and it was encrypted using the previous DEK, we need to wait to trigger a snapshot until we write and apply WAL `i+1`.

`manager/state/raft/storage/storage.go`'s `EncryptedRaftLogger` manages reading and writing WAL and snapshots.  Reading, writing, and switching out encryption keys all require a lock on the `EncryptedRaftLogger` object.

Keeping track of the last WAL index written using the old raft DEK, triggering a new snapshot when the applied index is higher than the last WAL index, and finishing the raft DEK rotation, is the job of `manager/state/raft/raft.go`'s `Node` object.

It's possible a manager may die or be shut down while in the middle of the DEK rotation process, which can take a little while due to waiting for the next WAL.  That is why both the current and pending DEK must be written to disk before the re-encryption process enumerated above begins.  Once the snapshot with index `i` (or `i+1`, if there was already a previous snapshot) is written, then the rotation process can be completed, and the current DEK replaced with the pending DEK, and the pending DEK deleted entirely.

In addition, it is possible that re-encrypting all raft data may already be in process when another raft DEK rotation is scheduled.  Consider the case, for example, if a manager node has been down for a little while, and in the meanwhile autolock has been disabled and re-enabled again, specifically to trigger raft log re-encryption (for example, if the unlock key and the TLS key of one manager node were leaked, which would mean that the raft DEK for that node would be compromised).  We do not want to require that all managers finish DEK rotation before allowing a change in auto-lock status, since that means that a single manager node being down would mean that credentials could not be rotated.

In such a case, we write a flag to the TLS key that indicates that another rotation is needed.  When the re-encryption and DEK rotation has finished, if the flag is set, then a new pending DEK is generated and the process begins all over again.  This way, no matter how many times a raft DEK rotation is triggered while another is in progress, only one additional rotation will be performed.

## KeyReadWriter/RaftDEKManager and TLS key headers

Because the TLS key and the raft DEK should be encrypted using the same KEK (key encrypting key) - the unlock key - we need to make sure that they are written to atomically so that it is impossible to have a raft DEK encrypted with a different KEK than a TLS key.  That is why all reads and writes of the TLS key are done through the `KeyReadWriter`.  This utility locks all access to the TLS key and cert, so that all writes to the headers and all changes to the key itself must be serial.  It also handles rotation of the KEK (the unlock key), so that everything is encrypted using the same key all at once.

`KeyReadWriter`'s writing functionality can be called in three different cases:

1.  When the TLS key is rotated, which could be due to certificate expiry or CA rotation, or possibly after the cluster goes from non-autolocked to autolocked.  This is handled by the certificate renewal loop in `node/node.go`.
1.  When the unlock key (KEK) is rotated - the TLS key material itself as well as the raft DEK headers must be re-encrypted.  This is handled by `RaftDEKManager`'s `MaybeUpdateKEK` function, which is called whenever the manager main loop in `manager/manager.go` notices a cluster change that involves the unlock key.
1.  When the raft DEK rotation process proceeds a step (i.e. when a pending DEK needs to be written, deleted, or swapped with the current DEK).  This is handled by `RaftDEKManager` whenever the raft node succeeds in re-encrypting all the raft data.

These events can all happen independently or in combination simultaneously, and are each controlled by separate processes/event loops.  Furthermore, the utility that encrypts the TLS key should not necessarily know about specific process of DEK rotation, which happens only on managers and not workers.  That is why `KeyReadWriter` is abstracted from the raft DEK management.  `KeyReadWriter` accepts an optional `PEMKeyHeaders` interface, that is called with the current headers, the current KEK, and returns a new set of headers.  On workers, for instance, all headers are deleted, since no DEK headers are necessary.

`manager/deks.go`'s `RaftDEKManager` knows how to serialize/encrypt and de-serialize/decrypt the raft headers, and perform a DEK rotation (generating a new pending DEK, replacing the current DEK, etc).  The supported headers are:

- `raft-dek`: the current raft DEK
- `raft-dek-pending`: the pending raft DEK
- `raft-dek-needs-rotation`: whether another DEK rotation is needed
- `kek-version`: this is actually implemented in `KeyReadWriter`, but is used by `RaftDEKManager` to determine whether or not the KEK needs rotation


## Sample workflows

Here are some sample workflows for how and when the individual keys are encrypted and rotated.

### Cluster starts out auto-locked (i.e. with an unlock-key), and 2 managers join.

1. Cluster is bootstrapped with autolock enabled.
1. Leader is started, and an unlock key is automatically generated and added to the cluster object.  This key is displayed to the user so that they can use it to unlock managers in the future.
1. Leader generates a root CA (the key for which is never written to disk) and its own TLS key and certificate.  The TLS key is written to disk encrypted using the generated unlock key.
1. Leader generates its unique raft DEK (data encryption key), which is encrypted with the generated unlock key and written as a header in the TLS key.  All WALs and snapshots will be written with this DEK.
1. 2 other managers are joined to the cluster - the 2 other managers get their TLS certificates, request the unlock key from the CA when they request their certificates, and write the TLS key to disk encrypted with the unlock key (so the new managers will never write the TLS key to disk unencrypted, so long as the cluster is auto-locked).
1. When the new managers then join the raft cluster using their new TLS certificates, they generate their own DEKs and write the encrypted DEK (encrypted with the unlock key) as a header in their TLS keys.  They receive raft data from the leader, unencrypted except via mTLS, and start writing raft data to disk encrypted with the DEK.
1. One of the managers is rebooted - when it comes back up, it cannot rejoin the cluster because its TLS key and its raft logs are encrypted.
1. A user manually unlocks the cluster by providing the unlock key via a CLI command.
1. The TLS key can now be is decrypted and used to connect to the other managers.  The raft DEK is decrypted at the same time, and the manager uses it to decrypt its raft logs.  The manager then can rejoin the cluster, catching up on any missed logs including any key rotation events (which will cause the manager to re-encrypt the TLS key and raft DEK using the new unlock key).

### A running, auto-locked cluster has its unlock key rotated to a new unlock-key

1. A cluster with 3 managers that are autolocked has its unlock key rotated, possibly due to compromise (e.g. accidental posting to github) - an API request is made to rotate the unlock key.  The leader which handles the API request generates a new unique unlock key, and writes the new unlock key to the raft store.
1. Each manager, including the leader, is watching the raft store, and as the change is propagated via raft to each manager, they each re-encrypt their TLS key and their raft DEK (and any pending raft DEKS) and write all keys to `swarm/certificates/swarm-node.key` in a single, atomic write.
1. On reboot, each manager will now require the new unlock-key to restart.
1. As a note, the unlock key could have been rotated while one of the managers was down.  In this case, unlocking this manager would require the old unlock key, but as soon as it’s unlocked it can catch up and get the new key to use for encryption, and on the next restart, it will require the new unlock key.

### A running, auto-locked cluster has auto-locking disabled

Perhaps the administrator decides that manually unlocking each manager is too much trouble and the managers are running using FDE on highly secured machines anyway.

1.  A cluster with 3 managers that are autolocked has its auto-lock setting disabled.  The leader which handles this API request deletes the unlock-key from the raft store.
1.  Each manager, including the leader, is watching the raft store, and as the change is propagated via raft to each manager, they each decrypt their TLS key and their raft DEK (and any pending raft DEKs) and write all keys to `swarm/certificates/swarm-node.key` in a single, atomic write.
1.  On reboot, each manager can use its unencrypted TLS key to connect to other managers, and use its unencrypted raft DEK to decrypt its raft logs.
1.  As a note, the unlock key could have been removed while one of the managers was down.  In this case, unlocking this manager would require the old unlock key, but as soon as it’s unlocked it can catch up, see that the unlock key has been deleted, and on the next restart, it will no longer require any unlock key.

### A running, non-autolocked cluster has auto-locking enabled

1. A cluster with 3 managers that are running without auto-lock enabled.  Each one has its own TLS key written to disk unencrypted, along with the unencrypted raft DEK header, because even if autolock is disabled, the raft logs are still encrypted.
1. An API request comes in to auto-lock the cluster.  The leader which handles this API request generates a unique unlock key, and writes the new unlock key to the raft store.
1. Each manager, including the leader, is watching the raft store, and as the change is propagated via raft to each manager, they each re-encrypt their TLS key and their raft DEK using the new unlock key and write both to `swarm/certificates/swarm-node.key` in a single, atomic write.  In addition, that write will contain enough information for a raft DEK rotation:
    - a new unique raft DEK is generated, enrypted using the new unlock key, and written in the pending raft DEK TLS header
    - if there was already a pending raft DEK (meaning a rotation was already in progress), it had been unencrypted - we will re-encrypt it, and add a TLS header indicating that we need another rotation after the current pending rotation has finished.  This flag is not encrypted.
1. Each manager kicks off a DEK rotation (please see the section on DEK rotation) and a TLS key rotation (the manager requests a new TLS key and cert) in order to replace the credentials that were previously available in plaintext.  These may take a little while, so they happen asynchronously.
