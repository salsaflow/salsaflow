INSTALL=go install
GODEP_INSTALL=godep $(INSTALL)

TEST=go test -v
GODEP_TEST=godep $(TEST)

install:
	${INSTALL} github.com/salsaflow/salsaflow
	${INSTALL} github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
	${INSTALL} github.com/salsaflow/salsaflow/bin/hooks/salsaflow-post-checkout
	${INSTALL} github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push

test:
	${TEST} github.com/salsaflow/salsaflow/github/issues

godep-install:
	${GODEP_INSTALL} github.com/salsaflow/salsaflow
	${GODEP_INSTALL} github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
	${GODEP_INSTALL} github.com/salsaflow/salsaflow/bin/hooks/salsaflow-post-checkout
	${GODEP_INSTALL} github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push

godep-test:
	${GODEP_TEST} github.com/salsaflow/salsaflow/github/issues

deps.fetch:
	@cat Godeps/Godeps.json | \
		grep ImportPath | \
		tail -n +2 | \
		awk '{ print $$2 }' | \
		tr -d '",' | \
		xargs go get -d -u

deps.save:
	godep save ./...

deps.restore:
	godep restore

format:
	go fmt ./...
