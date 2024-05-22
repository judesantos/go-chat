#! /bin/sh

#
# Development setup script.
#
# Run once when initially cloning the project onto a development machine.
#

echo
echo "Setting up development environment for go-chat"

## Install docker
echo
echo "Setting up docker..."
sh ./development/setup-docker.sh

## Install, run docker containers
echo
echo "Setting up Redis, Pgsql..."

docker-compose up -d

## Install Go. Install project go libraries
echo
echo "Setting up Go..."
sh ./development/setup-go.sh

## Setup pgsql db
echo
echo "Setting up pgsql..."
go run ./development/setup-db.go

RET=$?
if [ $RET -eq 0 ]; then
  echo "Setup pgqsl success!"
else
  echo
  echo "Setup failed!"
  exit -1
fi

echo
echo "Installing go project library dependencies..."
go mod tidy -v

echo
echo "Dev. setup is complete!"
echo
