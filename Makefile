VERSION="0.1.4"
EXTERNAL_TOOLS=\
	github.com/mitchellh/gox \
	github.com/jteeuwen/go-bindata/...

default: test

# bin generates the binaries for all platforms.
bin: generate
	@sh -c "'${CURDIR}/scripts/build.sh'"

# dev creates binares for testing locally - they are put into ./bin and $GOPATH.
dev: generate
	@DEV=1 sh -c "'${CURDIR}/scripts/build.sh'"

# dist creates the binaries for distibution.
dist: generate
	@sh -c "'${CURDIR}/scripts/dist.sh' '${VERSION}'"

# updatedeps installs all the dependencies needed to run and build.
updatedeps:
	@sh -c "'${CURDIR}/scripts/deps.sh'"

# generate runs `go generate` to build the dynamically generated source files.
generate:
	@echo "==> Generating..."
	@find . -type f -name '.DS_Store' -delete
	@go list ./... \
		| grep -v "/vendor/" \
		| xargs -n1 go generate

# bootstrap installs the necessary go tools for development/build.
bootstrap:
	@echo "==> Bootstrapping..."
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing "$$t"" ; \
		go get -u "$$t"; \
	done

.PHONY: default bin dev dist updatedeps generate bootstrap
