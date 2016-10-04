#!/bin/bash
go test -v github.com/awgh/bencrypt/bc
go test -v github.com/awgh/bencrypt/ecc
go test -v github.com/awgh/bencrypt/rsa
go test -v github.com/awgh/ratnet/nodes/qldb
go test -v github.com/awgh/ratnet/nodes/ram
go test -v github.com/awgh/ratnet/ratnet
