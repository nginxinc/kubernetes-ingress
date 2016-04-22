all: push

TAG = 0.1
PREFIX = gcr.io/nginx-ingress/nginx-plus-ingress

nginx-plus-ingress:
	CGO_ENABLED=0 GOOS=linux godep go build -a -installsuffix cgo -ldflags '-w' -o nginx-plus-ingress *.go

test:
	godep go test ./...

container: nginx-plus-ingress test
	docker build -t $(PREFIX):$(TAG) .

push: container
	gcloud docker push $(PREFIX):$(TAG)

osx:
	CGO_ENABLED=0 GOOS=darwin godep go build -a -installsuffix cgo -ldflags '-w' -o osx-nginx-plus-ingress *.go

clean:
	rm -f nginx-plus-ingress
