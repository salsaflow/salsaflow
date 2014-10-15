install: format
	go install github.com/salsita/salsaflow
	go install github.com/salsita/salsaflow/bin/hooks/salsaflow-commit-msg
	go install github.com/salsita/salsaflow/bin/hooks/salsaflow-pre-push

update-deps:
	go get -d -u bitbucket.org/kardianos/osext
	go get -d -u code.google.com/p/goauth2/oauth
	go get -d -u github.com/coreos/go-semver/semver
	go get -d -u github.com/extemporalgenome/slug
	go get -d -u github.com/google/go-github/github
	go get -d -u github.com/google/go-querystring/query
	go get -d -u gopkg.in/tchap/gocli.v1
	go get -d -u gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal
	go get -d -u gopkg.in/yaml.v2

format:
	go fmt ./...
