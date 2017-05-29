#!/bin/sh
docker run --name kdbs -v ~/projects/q/data:/data -w /data -v /db -p 5042:6020 -it derekwisong/kdb-server
docker run --name kdb -v ~/projects/q/data:/data -w /data -v ~/projects/q/db:/db --link kdbs -it derekwisong/kdb

