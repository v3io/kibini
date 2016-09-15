SHELL := /bin/bash
.DEFAULT_GOAL := kibini

${GOPATH}/bin/gox:
	go install ./vendor/github.com/mitchellh/gox

.PHONY: clean
clean:
	rm -rf ${GOPATH}/bin/*

.PHONY: kibini
kibini: ${GOPATH}/bin/gox
	@for osarch in "darwin/amd64" "linux/amd64"; do \
		cd ${GOPATH}/bin && ./gox -osarch="$$osarch" -output {{.OS}}_{{.Arch}}/{{.Dir}} ../src/github.com/iguazio/kibini; \
	done
