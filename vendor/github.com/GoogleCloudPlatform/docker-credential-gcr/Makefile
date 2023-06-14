# The file name of the binary to output
BINARY_FILENAME := docker-credential-gcr
# The output directory
OUT_DIR := bin
# The directory to dump generated mocks
MOCK_DIR := mock
# The non-vendor golang sources to validate.
SRCS = $(go list ./... | grep -v vendor)

all: clean bin

deps:
	@go get -u -t ./...

bin: deps
	@go build -i -o ${OUT_DIR}/${BINARY_FILENAME} main.go
	@echo Binary created: ${OUT_DIR}/${BINARY_FILENAME}

test-bin: deps
	@go build -i -o test/testdata/docker-credential-gcr main.go

clean:
	@rm -rf ${OUT_DIR}
	@go clean

# This re-generates the mocks using mockgen. Use this if tests don't compile due to type errors with
# the existing mocks.
mocks:
	@go get -u github.com/golang/mock/mockgen
	@rm -rf ${MOCK_DIR}
	@mkdir -p ${MOCK_DIR}/mock_store
	@mkdir -p ${MOCK_DIR}/mock_config
	@mkdir -p ${MOCK_DIR}/mock_cmd
	@mockgen -destination ${MOCK_DIR}/mock_store/mocks.go github.com/GoogleCloudPlatform/docker-credential-gcr/store GCRCredStore
	@mockgen -destination ${MOCK_DIR}/mock_config/mocks.go github.com/GoogleCloudPlatform/docker-credential-gcr/config UserConfig
	@mockgen -destination ${MOCK_DIR}/mock_cmd/mocks.go github.com/GoogleCloudPlatform/docker-credential-gcr/util/cmd Command
# mockgen doesn't play nice with vendor: https://github.com/golang/mock/issues/30
# The differences in -i's behavior on OSX and linux necessitate the creation
# of .bak files, which we want to clean up afterward...
	@find ${MOCK_DIR} -name '*.go' -exec sed -i.bak -e 's,github.com/GoogleCloudPlatform/docker-credential-gcr/vendor/,,g' {} \;
	@find ${MOCK_DIR} -name '*.go.bak' -exec rm {} \;

test: clean deps test-bin
	@go test -timeout 10s -v ./...

tests-unit: deps
	@go test -race -timeout 10s -v -tags=unit ./...

vet:
	@go vet ./...

lint:
	@echo 'Running golint...'
	@$(foreach src,$(SRCS),golint $(src);)

criticism: clean vet lint

fmt:
	@gofmt -w -s .

fix:
	@go fix ./...

pretty: fmt fix

presubmit: deps criticism pretty bin tests-unit test
