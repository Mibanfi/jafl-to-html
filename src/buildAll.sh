#!/bin/sh
echo "Building for linux..."
go build -o ../release/jaflToHtml-linux jaflToHtml.go
chmod +x ../release/jaflToHtml-linux
echo "completed"
echo "Building for windows..."
GOOS=windows GOARCH=amd64 go build -o ../release/jaflToHtml-win.exe jaflToHtml.go
echo "completed"
echo "Building for mac-amd64..."
GOOS=darwin GOARCH=amd64 go build -o ../release/jaflToHtml-mac-amd64 jaflToHtml.go
echo "completed"
echo "Building for mac-arm64..."
GOOS=darwin GOARCH=arm64 go build -o ../release/jaflToHtml-mac-arm64 jaflToHtml.go
echo "completed"
