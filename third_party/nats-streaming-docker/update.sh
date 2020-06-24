#!/bin/bash

set -e

if [ $# -eq 0 ] ; then
	echo "Usage: ./update.sh <nats-io/nats-streaming-server tag or branch>"
	exit
fi

VERSION=$1

# cd to the current directory so the script can be run from anywhere.
cd `dirname $0`

echo "Fetching and building nats-streaming-server $VERSION..."

# Create a tmp build directory.
TEMP=/tmp/nats-streaming.build
mkdir $TEMP

git clone -b $VERSION https://github.com/nats-io/nats-streaming-server $TEMP

docker build -t nats-streaming-builder $TEMP

# Create a dummy nats streaming builder container so we can run a cp against it.
ID=$(docker create nats-streaming-builder)

# Update the local binaries.
docker cp $ID:/go/src/github.com/nats-io/nats-streaming-server/pkg/linux-amd64/nats-streaming-server amd64/
#docker cp $ID:/go/src/github.com/nats-io/nats-streaming-server/pkg/linux-arm6/nats-streaming-server arm32v6/
#docker cp $ID:/go/src/github.com/nats-io/nats-streaming-server/pkg/linux-arm7/nats-streaming-server arm32v7/
#docker cp $ID:/go/src/github.com/nats-io/nats-streaming-server/pkg/linux-arm64/nats-streaming-server arm64v8/
#docker cp $ID:/go/src/github.com/nats-io/nats-streaming-server/pkg/win-amd64/nats-streaming-server.exe windows/nanoserver-1809/
#docker cp $ID:/go/src/github.com/nats-io/nats-streaming-server/pkg/win-amd64/nats-streaming-server.exe windows/windowsservercore/

# Cleanup.
rm -fr $TEMP
docker rm -f $ID
docker rmi nats-streaming-builder

echo "Done."
