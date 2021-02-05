DOCKER_TAG=ghcr.io/lindluni/runner-virtualbox:0.0.1

.PHONY: docker
docker:
	docker build --rm -t $(DOCKER_TAG) .

.PHONY: client
client:
	cd client && go build -o virtualbox-client .

.PHONY: server
server:
	cd server && go build -o virtualbox-server .