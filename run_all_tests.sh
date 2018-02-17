#!/bin/bash

rm -f nodes/qldb/tmp/ratnet_test.ql
rm -f ratnet/tmp/ratnet_p2p_test1.ql
rm -f ratnet/tmp/ratnet_p2p_test2.ql
rm -f ratnet/tmp/ratnet_test1.ql
rm -rf nodes/fs/tmp

go test -v github.com/awgh/bencrypt/bc
go test -v github.com/awgh/bencrypt/ecc
go test -v github.com/awgh/bencrypt/rsa
go test -v github.com/awgh/ratnet/nodes/qldb
go test -v github.com/awgh/ratnet/nodes/ram
go test -v github.com/awgh/ratnet/nodes/fs
go test -v github.com/awgh/ratnet/ratnet
