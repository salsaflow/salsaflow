install: format
	go install github.com/salsita/salsaflow
	go install github.com/salsita/salsaflow/bin/hooks/salsaflow-commit-msg

update-deps:
	go get -u bitbucket.org/kardianos/osext
	go get -u github.com/extemporalgenome/slug
	go get -u github.com/google/go-querystring/query
	go get -u gopkg.in/tchap/gocli.v1
	go get -u gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal
	go get -u gopkg.in/yaml.v2

format:
	go fmt ./...
