default: build

.PHONY: build
build:
	go build -o terraform-provider-appdynamics.exe .

.PHONY: install
install: build
	go build -o "$(shell go env GOPATH)/bin/terraform-provider-appdynamics.exe" .

.PHONY: fmt
fmt:
	gofmt -w -s .

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./... -v

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
