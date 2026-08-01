.PHONY: fmt test vet check player build-windows

fmt:
	gofmt -w *.go

test:
	go test -race ./...

vet:
	go vet ./...

check: fmt vet test

player:
	./player/build.sh

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H windowsgui -s -w" -o "GBA Video Maker.exe" .
	go run ./tools/embedicon -exe "GBA Video Maker.exe" -ico "assets/app_icon.ico"
