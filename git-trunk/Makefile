install: format
	go install github.com/salsita/SalsaFlow/git-trunk
	go install github.com/salsita/SalsaFlow/git-trunk/bin/hooks/commit-msg

update-deps:
	go get -u gopkg.in/tchap/gocli.v1
	go get -u gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal

format:
	go fmt ./...
