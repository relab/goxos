#!/bin/bash

#----------------------------------------------------------------------------
# Download and extract specific Go version
#----------------------------------------------------------------------------

go_version=go1.4.linux-amd64
echo Attempting to use Go version $go_version

if [ ! -e $go_version.tar.gz ]
then
	rm -rf $WORKSPACE/go
	echo Downloading $go_version
	wget https://storage.googleapis.com/golang/$go_version.tar.gz
	tar xf $go_version.tar.gz
fi

#----------------------------------------------------------------------------
# Environment variables
#----------------------------------------------------------------------------

echo
echo Setting GOROOT...
export GOROOT=$WORKSPACE/go
echo GOROOT is $GOROOT

echo
echo Adding GOROOT/bin to PATH...
export PATH=$GOROOT/bin:$PATH

echo
echo Setting GOPATH...
export GOPATH=$WORKSPACE/gopath
echo GOPATH is $GOPATH

#----------------------------------------------------------------------------
# Check go folder structure
#----------------------------------------------------------------------------

echo
echo Checking go folder structure...
if [ ! -d "$GOPATH/src" ]; then
	echo Creating $GOPATH/src
	mkdir -p $GOPATH/src
fi
if [ ! -d "$GOPATH/bin" ]; then
	echo Creating $GOPATH/bin
	mkdir -p $GOPATH/bin
fi
if [ ! -d "$GOPATH/pkg" ]; then
	echo Creating $GOPATH/pkg
	mkdir -p $GOPATH/pkg
fi

#----------------------------------------------------------------------------
# go version
#----------------------------------------------------------------------------

echo
echo Printing version reported by the go tool...
go version

#----------------------------------------------------------------------------
# go build
#----------------------------------------------------------------------------

echo
echo Building repository...
go build -v github.com/relab/goxos/...
if [ $? -eq 1 ] ; then
	echo "ERROR: 'go build' failed..."
	exit 1
fi

#----------------------------------------------------------------------------
# go test
#----------------------------------------------------------------------------

echo
echo Running all tests...
go test github.com/relab/goxos/...
if [ $? -eq 1 ] ; then
	echo "ERROR: 'go test' failed..."
	exit 1
fi
