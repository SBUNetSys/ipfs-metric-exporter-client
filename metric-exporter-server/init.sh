#!/bin/sh
# init ipfs
chmod 755 -R ./bin
#cp ./bin/ipfs-v0.13.0-docker .
#readelf -a ipfs-v0.13.0-docker | grep NEEDED
export IPFS_PATH=/exporter-server/.ipfs
./bin/ipfs-v0.13.0-docker init
ls -la
mkdir ./.ipfs/plugins
cp ./bin/mexport-v0.13.0-docker.so ./.ipfs/plugins
python config.py

IPFS_FD_MAX="100000" ./bin/ipfs-v0.13.0-docker daemon 
# 2>&1 | grep -a "metric-export"
