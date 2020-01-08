#!/usr/bin/env bats

load helpers

function setup() {
	teardown_busybox
	setup_busybox
}

function teardown() {
	teardown_busybox
}

@test "runc run [tty ptsname]" {
	# Replace sh script with readlink.
    sed -i 's|"sh"|"sh", "-c", "for file in /proc/self/fd/[012]; do readlink $file; done"|' config.json

	# run busybox
	runc run test_busybox
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ /dev/pts/+ ]]
	[[ ${lines[1]} =~ /dev/pts/+ ]]
	[[ ${lines[2]} =~ /dev/pts/+ ]]
}

@test "runc run [tty owner]" {
	# tty chmod is not doable in rootless containers without idmap.
	# TODO: this can be made as a change to the gid test.
	[[ "$ROOTLESS" -ne 0 ]] && requires rootless_idmap

	# Replace sh script with stat.
	sed -i 's/"sh"/"sh", "-c", "stat -c %u:%g $(tty) | tr : \\\\\\\\n"/' config.json

	# run busybox
	runc run test_busybox
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ 0 ]]
	# This is set by the default config.json (it corresponds to the standard tty group).
	[[ ${lines[1]} =~ 5 ]]
}

@test "runc run [tty owner] ({u,g}id != 0)" {
	# tty chmod is not doable in rootless containers without idmap.
	[[ "$ROOTLESS" -ne 0 ]] && requires rootless_idmap

	# replace "uid": 0 with "uid": 1000
	# and do a similar thing for gid.
	sed -i 's;"uid": 0;"uid": 1000;g' config.json
	sed -i 's;"gid": 0;"gid": 100;g' config.json

	# Replace sh script with stat.
	sed -i 's/"sh"/"sh", "-c", "stat -c %u:%g $(tty) | tr : \\\\\\\\n"/' config.json

	# run busybox
	runc run test_busybox
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ 1000 ]]
	# This is set by the default config.json (it corresponds to the standard tty group).
	[[ ${lines[1]} =~ 5 ]]
}

@test "runc exec [tty ptsname]" {
	# run busybox detached
	runc run -d --console-socket $CONSOLE_SOCKET test_busybox
	[ "$status" -eq 0 ]

	# make sure we're running
	testcontainer test_busybox running

	# run the exec
    runc exec test_busybox sh -c 'for file in /proc/self/fd/[012]; do readlink $file; done'
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ /dev/pts/+ ]]
	[[ ${lines[1]} =~ /dev/pts/+ ]]
	[[ ${lines[2]} =~ /dev/pts/+ ]]
}

@test "runc exec [tty owner]" {
	# tty chmod is not doable in rootless containers without idmap.
	# TODO: this can be made as a change to the gid test.
	[[ "$ROOTLESS" -ne 0 ]] && requires rootless_idmap

	# run busybox detached
	runc run -d --console-socket $CONSOLE_SOCKET test_busybox
	[ "$status" -eq 0 ]

	# make sure we're running
	testcontainer test_busybox running

	# run the exec
    runc exec test_busybox sh -c 'stat -c %u:%g $(tty) | tr : \\n'
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ 0 ]]
	[[ ${lines[1]} =~ 5 ]]
}

@test "runc exec [tty owner] ({u,g}id != 0)" {
	# tty chmod is not doable in rootless containers without idmap.
	[[ "$ROOTLESS" -ne 0 ]] && requires rootless_idmap

	# replace "uid": 0 with "uid": 1000
	# and do a similar thing for gid.
	sed -i 's;"uid": 0;"uid": 1000;g' config.json
	sed -i 's;"gid": 0;"gid": 100;g' config.json

	# run busybox detached
	runc run -d --console-socket $CONSOLE_SOCKET test_busybox
	[ "$status" -eq 0 ]

	# make sure we're running
	testcontainer test_busybox running

	# run the exec
	runc exec test_busybox sh -c 'stat -c %u:%g $(tty) | tr : \\n'
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ 1000 ]]
	[[ ${lines[1]} =~ 5 ]]
}

@test "runc exec [tty consolesize]" {
	# allow writing to filesystem
	sed -i 's/"readonly": true/"readonly": false/' config.json

	# run busybox detached
	runc run -d --console-socket $CONSOLE_SOCKET test_busybox
	[ "$status" -eq 0 ]

	# make sure we're running
	testcontainer test_busybox running

	tty_info_with_consize_size=$( cat <<EOF
{
    "terminal": true,
    "consoleSize": {
	    "height": 10,
	    "width": 110
    },
    "args": [
	    "/bin/sh",
	    "-c",
	    "/bin/stty -a > /tmp/tty-info"
    ],
    "cwd": "/"
}
EOF
	)

	# run the exec
	runc exec --pid-file pid.txt -d --console-socket $CONSOLE_SOCKET -p <( echo $tty_info_with_consize_size ) test_busybox
	[ "$status" -eq 0 ]

	# check the pid was generated
	[ -e pid.txt ]

	#wait user process to finish
	timeout 1 tail --pid=$(head -n 1 pid.txt) -f /dev/null

	tty_info=$( cat <<EOF
{
    "args": [
	"/bin/cat",
	"/tmp/tty-info"
    ],
    "cwd": "/"
}
EOF
	)

	# run the exec
	runc exec -p <( echo $tty_info ) test_busybox
	[ "$status" -eq 0 ]

	# test tty width and height against original process.json
	[[ ${lines[0]} =~ "rows 10; columns 110" ]]
}

@test "runc create [terminal=false]" {
	# Disable terminal creation.
	sed -i 's|"terminal": true,|"terminal": false,|g' config.json
	# Replace sh script with sleep.
    sed -i 's|"sh"|"sleep", "1000s"|' config.json

	# Make sure that the handling of detached IO is done properly. See #1354.
	__runc create test_busybox

	# Start the command.
	runc start test_busybox
	[ "$status" -eq 0 ]

	testcontainer test_busybox running

	# Kill the container.
	runc kill test_busybox KILL
	[ "$status" -eq 0 ]
}

@test "runc run [terminal=false]" {
	# Disable terminal creation.
	sed -i 's|"terminal": true,|"terminal": false,|g' config.json
	# Replace sh script with sleep.
    sed -i 's|"sh"|"sleep", "1000s"|' config.json

	# Make sure that the handling of non-detached IO is done properly. See #1354.
	(
		__runc run test_busybox
	) &

	wait_for_container 15 1 test_busybox
	testcontainer test_busybox running

	# Kill the container.
	runc kill test_busybox KILL
	[ "$status" -eq 0 ]
}

@test "runc run -d [terminal=false]" {
	# Disable terminal creation.
	sed -i 's|"terminal": true,|"terminal": false,|g' config.json
	# Replace sh script with sleep.
    sed -i 's|"sh"|"sleep", "1000s"|' config.json

	# Make sure that the handling of detached IO is done properly. See #1354.
	__runc run -d test_busybox

	testcontainer test_busybox running

	# Kill the container.
	runc kill test_busybox KILL
	[ "$status" -eq 0 ]
}
