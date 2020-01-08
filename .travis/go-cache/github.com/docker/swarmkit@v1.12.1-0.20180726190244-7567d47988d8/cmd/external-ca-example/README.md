# External CA Example

To get started, build and install the go program in this directory:

```
$ go install github.com/docker/swarmkit/cmd/external-ca-example
```

Now, run `external-ca-example`:

```
$ external-ca-example
INFO[0000] Now run: swarmd -d . --listen-control-api ./swarmd.sock --external-ca protocol=cfssl,url=https://localhost:58631/sign
```

This command initializes a new root CA along with the node certificate for the
first manager in a new cluster and saves it to a `certificates` directory in
the current directory. It then runs an HTTPS server on a random available port
which handles signing certificate requests from your manager nodes.

The server will continue to run after it prints out an example command to start
a new `swarmd` manager. Run this command in the current directory. You'll now
have a new swarm cluster which is configured to use this external CA.

Try joining new nodes to your cluster. Change into a new, empty directory and
run `swarmd` again with an argument to join the previous manager node:

```
$ swarmd -d . --listen-control-api ./swarmd.sock --listen-remote-api 0.0.0.0:4343 --join-addr localhost:4242 --join-token ...
Warning: Specifying a valid address with --listen-remote-api may be necessary for other managers to reach this one.
```

If this new node does not block indefinitely waiting for a TLS certificate to
be issued then everything is working correctly. Congratulations!
