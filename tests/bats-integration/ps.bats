#!/usr/bin/env bats

load helpers

function setup() {
  teardown_busybox
  setup_busybox
}

function teardown() {
  teardown_busybox
}

@test "ps" {
  # ps is not supported, it requires cgroups
  requires root

  # start busybox detached
  runv run -d --console-socket $CONSOLE_SOCKET test_busybox
  [ "$status" -eq 0 ]

  # check state
  testcontainer test_busybox running

  runv ps test_busybox
  [ "$status" -eq 0 ]
  # skip [[ ${lines[0]} =~ UID\ +PID\ +PPID\ +C\ +STIME\ +TTY\ +TIME\ +CMD+ ]]
  # skip [[ "${lines[1]}" == *"$(id -un 2>/dev/null)"*[0-9]* ]]
}

@test "ps -f json" {
  # ps is not supported, it requires cgroups
  requires root

  # start busybox detached
  runv run -d --console-socket $CONSOLE_SOCKET test_busybox
  [ "$status" -eq 0 ]

  # check state
  testcontainer test_busybox running

  runv ps -f json test_busybox
  [ "$status" -eq 0 ]
  [[ ${lines[0]} =~ [0-9]+ ]]
}

@test "ps -e -x" {
  # ps is not supported, it requires cgroups
  skip "runv ps does not support ps options"
  requires root

  # start busybox detached
  runv run -d --console-socket $CONSOLE_SOCKET test_busybox
  [ "$status" -eq 0 ]

  # check state
  testcontainer test_busybox running

  runv ps test_busybox -e -x
  [ "$status" -eq 0 ]
  # skip [[ ${lines[0]} =~ \ +PID\ +TTY\ +STAT\ +TIME\ +COMMAND+ ]]
  # skip [[ "${lines[1]}" =~ [0-9]+ ]]
}
