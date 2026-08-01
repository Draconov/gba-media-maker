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

build-windows:
	rm -f "GBA Video Maker.exe"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-H windowsgui -s -w" -o "GBA Video Maker.exe" .
	go run ./tools/embedicon -exe "GBA Video Maker.exe" -ico "assets/app_icon.ico"
