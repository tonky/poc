#!/bin/bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o poc

# mkdir /mnt/db
# chmod 777 /mnt/db

docker-compose up --build
