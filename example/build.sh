#!/bin/bash

set -e
set -o xtrace

SRC_DIR=./example
APP_BUNDLE=example/ExampleDaemon.app
MACOS_DIR=${APP_BUNDLE}/Contents/MacOS
RESOURCES_DIR=${APP_BUNDLE}/Contents/Resources
LAUNCH_DAEMONS_DIR=${APP_BUNDLE}/Contents/Library/LaunchDaemons
CODESIGN_IDENTITY=${CODESIGN_IDENTITY:-"Mac Developer: Albin Kerouanton (UPTPV854JD)"}

build() {
    # Daemon and client binaries are the same -- compile once and copy the
    # daemon in the right folder.

    go build -o ${MACOS_DIR}/client ${SRC_DIR}
    codesign --force --verify --verbose --options=runtime \
        --identifier="com.foobar.client" \
        --sign="${CODESIGN_IDENTITY}" \
        ${MACOS_DIR}/client

    cp ${MACOS_DIR}/client ${RESOURCES_DIR}/com.foobar.daemon
    codesign --force --verify --verbose --options=runtime \
        --identifier="com.foobar.daemon" \
        --sign="${CODESIGN_IDENTITY}" \
        ${RESOURCES_DIR}/com.foobar.daemon
}

install_daemon() {
    sudo cp ${RESOURCES_DIR}/com.foobar.daemon /Library/PrivilegedHelperTools/com.foobar.daemon
    sudo cp ${MACOS_DIR}/../Library/LaunchDaemons/com.foobar.daemon.plist /Library/LaunchDaemons/com.foobar.daemon.plist
    sudo launchctl bootout system/com.foobar.daemon || true
    sudo launchctl bootstrap system /Library/LaunchDaemons/com.foobar.daemon.plist
}

[ -d ${APP_BUNDLE} ] && rm -rf ${APP_BUNDLE}
mkdir -p ${MACOS_DIR} ${RESOURCES_DIR} ${LAUNCH_DAEMONS_DIR}
cp ${SRC_DIR}/com.foobar.daemon.plist ${LAUNCH_DAEMONS_DIR}/com.foobar.daemon.plist

build
install_daemon
