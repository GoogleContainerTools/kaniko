BINARIES=bin/buildkitd bin/buildctl
BINARIES_EXTRA=bin/buildkitd.oci_only bin/buildkitd.containerd_only bin/buildctl-darwin bin/buildkitd.exe bin/buildctl.exe
DESTDIR=/usr/local

binaries: $(BINARIES)
binaries-all: $(BINARIES) $(BINARIES_EXTRA)

bin/buildctl-darwin: FORCE
	mkdir -p bin
	docker build --build-arg GOOS=darwin -t buildkit:buildctl-darwin --target buildctl -f ./hack/dockerfiles/test.Dockerfile --force-rm .
	( containerID=$$(docker create buildkit:buildctl-darwin noop); \
		docker cp $$containerID:/usr/bin/buildctl $@; \
		docker rm $$containerID )
	chmod +x $@

bin/%.exe: FORCE
	mkdir -p bin
	docker build -t buildkit:$*.exe --target $*.exe -f ./hack/dockerfiles/test.Dockerfile --force-rm .
	( containerID=$$(docker create buildkit:$*.exe noop); \
		docker cp $$containerID:/$*.exe $@; \
		docker rm $$containerID )
	chmod +x $@

bin/%: FORCE
	mkdir -p bin
	docker build -t buildkit:$* --target $* -f ./hack/dockerfiles/test.Dockerfile --force-rm .
	( containerID=$$(docker create buildkit:$* noop); \
		docker cp $$containerID:/usr/bin/$* $@; \
		docker rm $$containerID )
	chmod +x $@

install: FORCE
	mkdir -p $(DESTDIR)/bin
	install $(BINARIES) $(DESTDIR)/bin

clean: FORCE
	rm -rf ./bin

test:
	./hack/test

lint:
	./hack/lint

validate-vendor:
	./hack/validate-vendor

validate-generated-files:
	./hack/validate-generated-files

validate-all: test lint validate-vendor validate-generated-files

vendor:
	./hack/update-vendor

generated-files:
	./hack/update-generated-files

.PHONY: vendor generated-files test binaries binaries-all install clean lint validate-all validate-vendor validate-generated-files
FORCE:
