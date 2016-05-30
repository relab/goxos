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

echo Adding GOPATH/bin to PATH...
export PATH=$PATH:$GOPATH/bin
echo PATH is $PATH

echo Setting NIGHTLYEXPS...
export NIGHTLYEXPS=$GOPATH/src/github.com/relab/goxos/exp/nightlyexperiments
echo NIGHTLYEXPS is $NIGHTLYEXPS

echo Setting EXPRUN...
export EXPRUN=$GOPATH/src/github.com/relab/goxos/exp/exprun
echo EXPRUN is $EXPRUN

echo Setting EXPPATH...
export EXPPATH=/home/ansatt/jenkins/output/ 
echo EXPPATH is $EXPPATH

echo Setting REPORTPATH...
export REPORTPATH=/home/ansatt/jenkins/public_html/Reports/ 
echo REPORTPATH is $REPORTPATH

echo Setting HTMLPATH...
export HTMLPATH=/home/ansatt/jenkins/public_html/
echo HTMLPATH is $HTMLPATH

echo Setting EXPANALYZE_BIN
export EXPANALYZE_BIN=$GOPATH/bin/expanalyze
echo EXPANALYZE_BIN is $EXPANALYZE_BIN

echo Setting VGFONTPATH
export VGFONTPATH=$GOPATH/src/github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/vg/fonts
echo VGFONTPATH is $VGFONTPATH

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
# go version, get, install and test
#----------------------------------------------------------------------------

echo
echo Printing version reported by the go tool...
go version

echo Bulding and installing every binary in repository... 
go install github.com/relab/goxos/...
if [ $? -eq 1 ] ; then
	echo "ERROR: 'go install' failed..."
	exit 1
fi

echo
echo Running all tests...
go test github.com/relab/goxos/...
if [ $? -eq 1 ] ; then
	echo "ERROR: 'go test' failed..."
	exit 1
fi

#----------------------------------------------------------------------------
# Running experiments
#----------------------------------------------------------------------------

echo
echo Running every found experiment in NIGHTLYEXPS 
cd $NIGHTLYEXPS
for f in *.ini
do
	echo Running experiment $f
	exppinghosts -exp_path=$NIGHTLYEXPS/$f -w
	cd $EXPRUN
	fab run_exp:$NIGHTLYEXPS/$f,local_out=$EXPPATH
done

#----------------------------------------------------------------------------
# Analyzing experiments
#----------------------------------------------------------------------------

echo
echo Running expanalyzeall...
expanalyzeall \
	-exppath=$EXPPATH \
	-reportpath=$REPORTPATH \
	-htmlpath=$HTMLPATH \
	-expanalyze-bin-path=$EXPANALYZE_BIN \
/
