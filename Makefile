install: format
	go install github.com/salsaflow/salsaflow
	go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
	go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push

godep-install:
	godep go install github.com/salsaflow/salsaflow
	godep go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
	godep go install github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push

deps.fetch:
	go get -d -u bitbucket.org/kardianos/osext
	go get -d -u code.google.com/p/goauth2/oauth
	go get -d -u github.com/coreos/go-semver/semver
	go get -d -u github.com/extemporalgenome/slug
	go get -d -u github.com/google/go-github/github
	go get -d -u github.com/google/go-querystring/query
	go get -d -u github.com/salsita/go-pivotaltracker/v5/pivotal
	go get -d -u github.com/salsita/go-sprintly/sprintly
	go get -d -u github.com/toqueteos/webbrowser
	go get -d -u gopkg.in/tchap/gocli.v2
	go get -d -u gopkg.in/yaml.v2

deps.save:
	godep save ./...

deps.restore:
	godep restore

format:
	go fmt ./...
