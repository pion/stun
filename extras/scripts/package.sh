#!/bin/sh
#
# Debian 8: systemd
# Ubuntu 16.04: systemd
# Centos 7: systemd
#
if [ "$1" = "" ]
then
  echo "Usage: $0 <version> <iteration>"
  exit
fi

if [ "$2" = "" ]
then
  echo "Usage: $0 <version> <iteration>"
  exit
fi

VERSION=$1
ITERATION=$2

INSTALL_DIR=/usr/bin
LOG_DIR=/var/log/stund
CONFIG_DIR=/etc/stund
LOGROTATE_DIR=/etc/logrotate.d
DATA_DIR=/var/lib/stund
SCRIPT_DIR=/usr/lib/stund

SAMPLE_CONFIGURATION=extras/packaging/config.json
SYSTEMD_SCRIPT=extras/packaging/stund.service
POSTINSTALL_SCRIPT=extras/packaging/post_install.sh
PREINSTALL_SCRIPT=extras/packaging/pre_install.sh
POSTUNINSTALL_SCRIPT=extras/packaging/post_uninstall.sh
LOGROTATE=extras/packaging/logrotate

TMP_WORK_DIR=`mktemp -d`
TMP_BINARIES_DIR=`mktemp -d`
ARCH=amd64
NAME=stund
LICENSE=MIT
URL="https://github.com/ernado/stun"
MAINTAINER="ar@cydev.ru"
VENDOR=stund
DESCRIPTION="STUN server"

echo "Start packaging, version: $VERSION, iteration: $ITERATION"

# check_gopath checks the GOPATH env variable set
#check_gopath() {
#    [ -z "$GOPATH" ] &amp;&amp; echo "GOPATH is not set." && cleanup_exit 1
#    echo "GOPATH: $GOPATH"
#}

# cleanup_exit removes all resources created during the process and exits with
# the supplied returned code.
cleanup_exit() {
    rm -r $TMP_WORK_DIR
    rm -r $TMP_BINARIES_DIR
    exit $1
}

# make_dir_tree creates the directory structure within the packages.
make_dir_tree() {
    work_dir=$1

    mkdir -p $work_dir/$INSTALL_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to create install directory -- aborting."
        cleanup_exit 1
    fi
    mkdir -p $work_dir/$SCRIPT_DIR/scripts
    if [ $? -ne 0 ]; then
        echo "Failed to create script directory -- aborting."
        cleanup_exit 1
    fi
    mkdir -p $work_dir/$CONFIG_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to create configuration directory -- aborting."
        cleanup_exit 1
    fi
    mkdir -p $work_dir/$LOGROTATE_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to create logrotate directory -- aborting."
        cleanup_exit 1
    fi
}

# do_build builds the code. The version and commit must be passed in.
do_build() {
    echo "Start building binary"
    gox -os="linux" -arch="amd64" -output="$TMP_BINARIES_DIR/{{.OS}}-{{.Arch}}/{{.Dir}}" ./stund
    echo "Binary build completed successfully"
}

do_build

make_dir_tree $TMP_WORK_DIR

cp $TMP_BINARIES_DIR/linux-amd64/stund $TMP_WORK_DIR/$INSTALL_DIR/
if [ $? -ne 0 ]; then
    echo "Failed to copy binaries to packaging directory ($TMP_WORK_DIR/$INSTALL_DIR/) -- aborting."
    cleanup_exit 1
fi

echo "stund binary copied to $TMP_WORK_DIR/$INSTALL_DIR/"

cp $SYSTEMD_SCRIPT $TMP_WORK_DIR/$SCRIPT_DIR/scripts/stund.service
if [ $? -ne 0 ]; then
    echo "Failed to copy systemd script to packaging directory -- aborting."
    cleanup_exit 1
fi

echo "systemd script copied to $TMP_WORK_DIR/$SCRIPT_DIR/scripts"

COMMON_FPM_ARGS="\
-C $TMP_WORK_DIR \
--log error \
--version $VERSION \
--iteration $ITERATION \
--name $NAME \
--vendor $VENDOR \
--url $URL \
--category Network \
--license $LICENSE \
--maintainer $MAINTAINER \
--force \
--after-install $POSTINSTALL_SCRIPT \
--before-install $PREINSTALL_SCRIPT \
--after-remove $POSTUNINSTALL_SCRIPT \
--config-files $CONFIG_DIR \
--config-files $LOGROTATE_DIR "

rm -r ./PACKAGES
mkdir -p PACKAGES

echo "Start building deb package"

fpm -s dir -t deb $COMMON_FPM_ARGS --description "$DESCRIPTION" \
    --deb-compression bzip2 \
    -p PACKAGES/ \
    -a amd64 .

echo "Packaging complete!"

cleanup_exit 0