aws_account_id := 233868805618
aws_region := us-east-1
docker_tag := hackmit-playground

build:
	docker build -t $(docker_tag) .

push: build
	aws ecr get-login-password --region $(aws_region) | docker login --username AWS --password-stdin $(aws_account_id).dkr.ecr.$(aws_region).amazonaws.com
	docker tag hackmit-playground:latest $(aws_account_id).dkr.ecr.$(aws_region).amazonaws.com/hackmit-playground:latest
	docker push $(aws_account_id).dkr.ecr.$(aws_region).amazonaws.com/hackmit-playground:latest
