REPO = ctfsys
SVC_NAME = hackersvc

# Build a binary that runs on your machine
# Alias for bin/hackersvc
build: bin/hackersvc
.PHONY: build

# Create container image (docker)
# Alias for docker-build
image: docker-build
.PHONY: image

# Build a binary that runs on your machine
bin/$(SVC_NAME): $(SVC_NAME).go
	go build -o bin/$(SVC_NAME) $(SVC_NAME).go

# Build docker image
docker-build: build/Dockerfile image-linux
	docker build -f build/Dockerfile -t $(REPO)/$(SVC_NAME):$(TAG) .
.PHONY: docker-build

# Build binary that can run in our docker container
image-linux: build/bin/$(SVC_NAME)
build/bin/$(SVC_NAME): hackerservice.go
	GOOS=linux GOARCH=amd64 \
	go build -o build/bin/$(SVC_NAME) $(SVC_NAME).go
.PHONY: image-linux
