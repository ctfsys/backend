REPO = ctfsys
SVC_NAME = hackercli

# Build a binary that runs on your machine
# Alias for bin/hackercli
build: bin/hackercli
.PHONY: build

# Create container image (docker)
# Alias for docker-build
image: docker-build
.PHONY: image

# Build a binary that runs on your machine
bin/$(SVC_NAME): $(SVC_NAME).go
	go build -o bin/$(SVC_NAME) $(SVC_NAME).go

docker-build: build/Dockerfile image-linux
	docker build -f build/Dockerfile -t $(REPO)/$(SVC_NAME):$(TAG) .
.PHONY: docker-build

# Build a binary that can run in our docker container
image-linux: build/bin/$(SVC_NAME)
build/bin/$(SVC_NAME): hackerservice.go
	GOOS=linux GOARCH=amd64 \
	go build -o build/bin/$(SVC_NAME) $(SVC_NAME).go
