.PHONY: test vet build-windows package clean

test:
	go test ./...

vet:
	go vet ./...
	GOOS=windows GOARCH=amd64 go vet ./...

build-windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/skvoz.exe ./cmd/skvoz

package:
	scripts/package.sh $(VERSION)

clean:
	rm -rf dist
