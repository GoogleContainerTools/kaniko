#!/usr/bin/env bats

load helpers

function teardown() {
    rm -f $BATS_TMPDIR/runc-cgroups-integration-test.json
    teardown_running_container test_cgroups_kmem
    teardown_running_container test_cgroups_permissions
    teardown_busybox
}

function setup() {
    teardown
    setup_busybox
}

function check_cgroup_value() {
    cgroup=$1
    source=$2
    expected=$3

    current=$(cat $cgroup/$source)
    echo  $cgroup/$source
    echo "current" $current "!?" "$expected"
    [ "$current" -eq "$expected" ]
}

@test "runc update --kernel-memory (initialized)" {
    [[ "$ROOTLESS" -ne 0 ]] && requires rootless_cgroup
    requires cgroups_kmem

    set_cgroups_path "$BUSYBOX_BUNDLE"

    # Set some initial known values
    DATA=$(cat <<-EOF
    "memory": {
        "kernel": 16777216
    },
EOF
    )
    DATA=$(echo ${DATA} | sed 's/\n/\\n/g')
    sed -i "s/\(\"resources\": {\)/\1\n${DATA}/" ${BUSYBOX_BUNDLE}/config.json

    # run a detached busybox to work with
    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_kmem
    [ "$status" -eq 0 ]

    # update kernel memory limit
    runc update test_cgroups_kmem --kernel-memory 50331648
    [ "$status" -eq 0 ]

	# check the value
    check_cgroup_value $CGROUP_MEMORY "memory.kmem.limit_in_bytes" 50331648
}

@test "runc update --kernel-memory (uninitialized)" {
    [[ "$ROOTLESS" -ne 0 ]] && requires rootless_cgroup
    requires cgroups_kmem

    set_cgroups_path "$BUSYBOX_BUNDLE"

    # run a detached busybox to work with
    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_kmem
    [ "$status" -eq 0 ]

    # update kernel memory limit
    runc update test_cgroups_kmem --kernel-memory 50331648
    # Since kernel 4.6, we can update kernel memory without initialization
    # because it's accounted by default.
    if [ "$KERNEL_MAJOR" -lt 4 ] || [ "$KERNEL_MAJOR" -eq 4 -a "$KERNEL_MINOR" -le 5 ]; then
        [ ! "$status" -eq 0 ]
    else
        [ "$status" -eq 0 ]
        check_cgroup_value $CGROUP_MEMORY "memory.kmem.limit_in_bytes" 50331648
    fi
}

@test "runc create (no limits + no cgrouppath + no permission) succeeds" {
    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_permissions
    [ "$status" -eq 0 ]
}

@test "runc create (rootless + no limits + cgrouppath + no permission) fails with permission error" {
    requires rootless
    requires rootless_no_cgroup

    set_cgroups_path "$BUSYBOX_BUNDLE"

    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_permissions
    [ "$status" -eq 1 ]
    [[ ${lines[1]} == *"permission denied"* ]]
}

@test "runc create (rootless + limits + no cgrouppath + no permission) fails with informative error" {
    requires rootless
    requires rootless_no_cgroup

    set_resources_limit "$BUSYBOX_BUNDLE"

    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_permissions
    [ "$status" -eq 1 ]
    [[ ${lines[1]} == *"cannot set limits on the pids cgroup, as the container has not joined it"* ]]
}

@test "runc create (limits + cgrouppath + permission on the cgroup dir) succeeds" {
   [[ "$ROOTLESS" -ne 0 ]] && requires rootless_cgroup

    set_cgroups_path "$BUSYBOX_BUNDLE"
    set_resources_limit "$BUSYBOX_BUNDLE"

    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_permissions
    [ "$status" -eq 0 ]
}

@test "runc exec (limits + cgrouppath + permission on the cgroup dir) succeeds" {
   [[ "$ROOTLESS" -ne 0 ]] && requires rootless_cgroup

    set_cgroups_path "$BUSYBOX_BUNDLE"
    set_resources_limit "$BUSYBOX_BUNDLE"

    runc run -d --console-socket $CONSOLE_SOCKET test_cgroups_permissions
    [ "$status" -eq 0 ]

    runc exec test_cgroups_permissions echo "cgroups_exec"
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == *"cgroups_exec"* ]]
}
