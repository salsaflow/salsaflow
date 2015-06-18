install:
	go install github.com/salsaflow/salsaflow
	go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
	go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-post-checkout
	go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push

godep-install:
	godep go install github.com/salsaflow/salsaflow
	godep go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
	godep go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-post-checkout
	godep go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push

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
