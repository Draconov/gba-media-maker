.PHONY: fmt test vet check player build-windows

fmt:
	go fmt ./...

test:
	go test -race ./...

vet:
	go vet ./...

check: fmt vet test

player:
	bash player/build.sh

build-windows: player
	rm -f "GBA Media Maker.exe"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-H windowsgui -s -w" -o "GBA Media Maker.exe" .
	go run ./tools/embedicon -exe "GBA Media Maker.exe" -ico "assets/icon.ico"
