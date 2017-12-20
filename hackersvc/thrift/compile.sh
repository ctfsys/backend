#!/usr/bin/env sh

# See also https://thrift.apache.org/tutorial/go

thrift -r --gen "go:package_prefix=github.com/ctfsys/backend/hackersvc/thrift/gen-go/,thrift_import=github.com/apache/thrift/lib/go/thrift" hackersvc.thrift
