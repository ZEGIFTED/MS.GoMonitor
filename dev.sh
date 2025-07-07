#!/bin/bash

echo "------------------------- BUILDING MONITORING ENGINE ----------------------------"

echo "=> Cleaning Build Cache"
# go clean -cache -modcache

# Build plugins
echo "=> Building Plugins"
# go build -buildmode=plugin -o plugins/http_monitor.so ./pkg/plugins/http_monitor/
# go build -buildmode=plugin -o plugins/ssl_check.so ./pkg/plugins/ssl_check/
go build -buildmode=plugin -o plugins/response_time.so ./pkg/plugins/response_time/

go build -o ms_engine ./cmd/ms

# Run main application
echo "------------------------- RUNNING MONITORING ENGINE ----------------------------"
./ms_engine --plugin-dir ./plugins
# go run cmd/ms/main.go --plugin-dir ./plugins