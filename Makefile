APP := app
GOFMT := gofmt

.PHONY: dev test vet fmt lint build preview clean

dev:
	cd $(APP) && go tool mygo dev

test:
	cd $(APP) && CGO_ENABLED=0 go test ./...

vet:
	cd $(APP) && go vet ./... && go tool mygo vet .

fmt:
	cd $(APP) && $(GOFMT) -w .

lint:
	cd $(APP) && test -z "$$($(GOFMT) -l .)" && go vet ./... && go tool mygo vet . && go run honnef.co/go/tools/cmd/staticcheck@latest ./...

build:
	cd $(APP) && go tool mygo build

preview:
	python3 bin/preview_seed.py $(APP)/.preview
	cd $(APP) && TOKI_PREVIEW_DATA="$$PWD/.preview" TOKI_PREVIEW_OUT=../docs/screenshots \
		go test -run 'TestPreview|TestPreviewStates' -v

clean:
	rm -rf $(APP)/build $(APP)/.preview $(APP)/.mygo
