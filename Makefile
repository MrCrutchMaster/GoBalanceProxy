lint:
	golangci-lint -v run cmd/... pkg/...

emu_build:
	CGO_ENABLED=0 GOOS=linux go build  -mod vendor -a -installsuffix cgo  -ldflags "${LDFLAGS}" -o _bin/endpointEmulator ./_test/endpoints_emulator/*.go

emu_run:
	./_bin/endpointEmulator

build:
	export VERSION="1.0.0"
	export BUILDNUMBER=$(date +%Y%m%d.%H%M-%S)
	export LDFLAGS="-w -s -X main.buildNumber=${BUILDNUMBER} -X main.goVersion=1.20.4 -X main.version=${VERSION}"
	CGO_ENABLED=0 GOOS=linux go build  -mod vendor -a -installsuffix cgo  -ldflags "${LDFLAGS}" -o _bin/goBalancer ./cmd/*.go

run:
	./_bin/goBalancer -c ./_contrib/config.yaml

