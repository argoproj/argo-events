#!/bin/bash

set -e

if [ $# -eq 0 ] ; then
	echo "Usage: ./update.sh <nats-io/nats-streaming-server tag or branch>"
	exit
fi

VERSION=$1

cd `dirname $0`

echo "Downloading nats-streaming-server $VERSION..."

wget -O nats-streaming-server.tar.gz "https://github.com/nats-io/nats-streaming-server/releases/download/${VERSION}/nats-streaming-server-${VERSION}-linux-amd64.tar.gz" 
tar -xf nats-streaming-server.tar.gz
rm nats-streaming-server.tar.gz

echo "Done."
