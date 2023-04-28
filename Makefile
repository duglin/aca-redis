IMAGE=ghcr.io/duglin/aca-redis
IMAGE2=duglin/aca-redis

.image: app.go go.* Dockerfile
	go build -o /dev/null app.go	# quick fail
	docker build -t $(IMAGE) .
	docker push $(IMAGE)
	docker tag $(IMAGE) $(IMAGE2)
	docker push $(IMAGE2)
	touch .image

all: .image

