#!/usr/bin/env bats

load helpers

function setup() {
  teardown_busybox
  setup_busybox
}

function teardown() {
  teardown_busybox
}

@test "checkpoint and restore" {
  # XXX: currently criu require root containers.
  requires criu root

  # criu does not work with external terminals so..
  # setting terminal and root:readonly: to false

  runc run -d --console-socket $CONSOLE_SOCKET test_busybox
  [ "$status" -eq 0 ]

  testcontainer test_busybox running

  for i in `seq 2`; do
    # checkpoint the running container
    runc --criu "$CRIU" checkpoint --work-path ./work-dir test_busybox
    ret=$?
    # if you are having problems getting criu to work uncomment the following dump:
    #cat /run/opencontainer/containers/test_busybox/criu.work/dump.log
    cat ./work-dir/dump.log | grep -B 5 Error || true
    [ "$ret" -eq 0 ]

    # after checkpoint busybox is no longer running
    runc state test_busybox
    [ "$status" -ne 0 ]

    # restore from checkpoint
    runc --criu "$CRIU" restore -d --work-path ./work-dir --console-socket $CONSOLE_SOCKET test_busybox
    ret=$?
    cat ./work-dir/restore.log | grep -B 5 Error || true
    [ "$ret" -eq 0 ]

    # busybox should be back up and running
    testcontainer test_busybox running
  done
}

@test "checkpoint --pre-dump and restore" {
  # XXX: currently criu require root containers.
  requires criu root

  sed -i 's;"terminal": true;"terminal": false;' config.json
  sed -i 's;"readonly": true;"readonly": false;' config.json
  sed -i 's/"sh"/"sh","-c","for i in `seq 10`; do read xxx || continue; echo ponG $xxx; done"/' config.json

  # The following code creates pipes for stdin and stdout.
  # CRIU can't handle fifo-s, so we need all these tricks.
  fifo=`mktemp -u /tmp/runc-fifo-XXXXXX`
  mkfifo $fifo

  # stdout
  cat $fifo | cat $fifo &
  pid=$!
  exec 50</proc/$pid/fd/0
  exec 51>/proc/$pid/fd/0

  # stdin
  cat $fifo | cat $fifo &
  pid=$!
  exec 60</proc/$pid/fd/0
  exec 61>/proc/$pid/fd/0

  echo -n > $fifo
  unlink $fifo

    # run busybox (not detached)
  __runc run -d test_busybox <&60 >&51 2>&51
  [ $? -eq 0 ]

  testcontainer test_busybox running

  #test checkpoint pre-dump
  mkdir parent-dir
  runc --criu "$CRIU" checkpoint --pre-dump --image-path ./parent-dir test_busybox
  [ "$status" -eq 0 ]

  # busybox should still be running
  runc state test_busybox
  [ "$status" -eq 0 ]
  [[ "${output}" == *"running"* ]]

  # checkpoint the running container
  mkdir image-dir
  mkdir work-dir
  runc --criu "$CRIU" checkpoint --parent-path ./parent-dir --work-path ./work-dir --image-path ./image-dir test_busybox
  cat ./work-dir/dump.log | grep -B 5 Error || true
  [ "$status" -eq 0 ]

  # after checkpoint busybox is no longer running
  runc state test_busybox
  [ "$status" -ne 0 ]

  # restore from checkpoint
  __runc --criu "$CRIU" restore -d --work-path ./work-dir --image-path ./image-dir test_busybox <&60 >&51 2>&51
  ret=$?
  cat ./work-dir/restore.log | grep -B 5 Error || true
  [ $ret -eq 0 ]

  # busybox should be back up and running
  testcontainer test_busybox running

  runc exec --cwd /bin test_busybox echo ok
  [ "$status" -eq 0 ]
  [[ ${output} == "ok" ]]

  echo Ping >&61
  exec 61>&-
  exec 51>&-
  run cat <&50
  [ "$status" -eq 0 ]
  [[ "${output}" == *"ponG Ping"* ]]
}

@test "checkpoint --lazy-pages and restore" {
  # XXX: currently criu require root containers.
  requires criu root

  # check if lazy-pages is supported
  run ${CRIU} check --feature lazy_pages
  if [ "$status" -eq 1 ]; then
    # this criu does not support lazy migration; skip the test
    skip "this criu does not support lazy migration"
  fi

  sed -i 's;"terminal": true;"terminal": false;' config.json
  sed -i 's;"readonly": true;"readonly": false;' config.json
  sed -i 's/"sh"/"sh","-c","for i in `seq 10`; do read xxx || continue; echo ponG $xxx; done"/' config.json

  # The following code creates pipes for stdin and stdout.
  # CRIU can't handle fifo-s, so we need all these tricks.
  fifo=`mktemp -u /tmp/runc-fifo-XXXXXX`
  mkfifo $fifo

  # For lazy migration we need to know when CRIU is ready to serve
  # the memory pages via TCP.
  lazy_pipe=`mktemp -u /tmp/lazy-pipe-XXXXXX`
  mkfifo $lazy_pipe

  # TCP port for lazy migration
  port=27277

  # stdout
  cat $fifo | cat $fifo &
  pid=$!
  exec 50</proc/$pid/fd/0
  exec 51>/proc/$pid/fd/0

  # stdin
  cat $fifo | cat $fifo &
  pid=$!
  exec 60</proc/$pid/fd/0
  exec 61>/proc/$pid/fd/0

  echo -n > $fifo
  unlink $fifo

  # run busybox
  __runc run -d test_busybox <&60 >&51 2>&51
  [ $? -eq 0 ]

  testcontainer test_busybox running

  # checkpoint the running container
  mkdir image-dir
  mkdir work-dir
  # Double fork taken from helpers.bats
  # We need to start 'runc checkpoint --lazy-pages' in the background,
  # so we double fork in the shell.
  (runc --criu "$CRIU" checkpoint --lazy-pages --page-server 0.0.0.0:${port} --status-fd ${lazy_pipe} --work-path ./work-dir --image-path ./image-dir test_busybox & ) &
  # Sleeping here. This is ugly, but not sure how else to handle it.
  # The return code of the in the background running runc is needed, if
  # there is some basic error. If the lazy migration is ready can
  # be handled by $lazy_pipe. Which probably will always be ready
  # after sleeping two seconds.
  sleep 2
  # Check if inventory.img was written
  [ -e image-dir/inventory.img ]
  # If the inventory.img exists criu checkpointed some things, let's see
  # if there were other errors in the log file.
  run grep -B 5 Error ./work-dir/dump.log -q
  [ "$status" -eq 1 ]

  # This will block until CRIU is ready to serve memory pages
  cat $lazy_pipe
  [ "$status" -eq 1 ]

  unlink $lazy_pipe

  # Double fork taken from helpers.bats
  # We need to start 'criu lazy-pages' in the background,
  # so we double fork in the shell.
  # Start CRIU in lazy-daemon mode
  $(${CRIU} lazy-pages --page-server --address 127.0.0.1 --port ${port} -D image-dir &) &

  # Restore lazily from checkpoint.
  # The restored container needs a different name as the checkpointed
  # container is not yet destroyed. It is only destroyed at that point
  # in time when the last page is lazily transferred to the destination.
  # Killing the CRIU on the checkpoint side will let the container
  # continue to run if the migration failed at some point.
  __runc --criu "$CRIU" restore -d --work-path ./image-dir --image-path ./image-dir --lazy-pages test_busybox_restore <&60 >&51 2>&51
  ret=$?
  [ $ret -eq 0 ]
  run grep -B 5 Error ./work-dir/dump.log -q
  [ "$status" -eq 1 ]

  # busybox should be back up and running
  testcontainer test_busybox_restore running

  runc exec --cwd /bin test_busybox_restore echo ok
  [ "$status" -eq 0 ]
  [[ ${output} == "ok" ]]

  echo Ping >&61
  exec 61>&-
  exec 51>&-
  run cat <&50
  [ "$status" -eq 0 ]
  [[ "${output}" == *"ponG Ping"* ]]
}
