docker_tag := hackmit-playground

build:
	docker build -t $(docker_tag) .

push: build
	docker tag hackmit-playground:latest 233868805618.dkr.ecr.us-east-1.amazonaws.com/hackmit-playground:latest
	docker push 233868805618.dkr.ecr.us-east-1.amazonaws.com/hackmit-playground:latest
