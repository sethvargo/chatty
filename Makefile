VERSION="0.1.1"
EXTERNAL_TOOLS=\
	github.com/mitchellh/gox

default: test

# bin generates the binaries for all platforms.
bin:
	@sh -c "'${CURDIR}/scripts/build.sh'"

# dev creates binares for testing locally - they are put into ./bin and $GOPATH.
dev:
	@DEV=1 sh -c "'${CURDIR}/scripts/build.sh'"

# dist creates the binaries for distibution.
dist:
	@sh -c "'${CURDIR}/scripts/dist.sh' '${VERSION}'"

# updatedeps installs all the dependencies needed to run and build.
updatedeps:
	@sh -c "'${CURDIR}/scripts/deps.sh'"

# bootstrap installs the necessary go tools for development/build.
bootstrap:
	@echo "==> Bootstrapping..."
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing "$$t"..." ; \
		go get -u "$$t"; \
	done

.PHONY: default bin dev dist updatedeps bootstrap
