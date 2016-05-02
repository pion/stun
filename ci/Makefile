pipeline:
	fly -t ci sp -p stun -c pipeline.yml --load-vars-from ~/.concourse/credentials.yml
unpause:
	fly -t ci up -p stun
resource:
	@echo "forcing dev branch to update on ci"
	@fly -t ci check-resource -r stun/dev
docker:
	sudo docker build -t cydev/gometalinter:latest images/gometalinter
docker-push:
	sudo docker push cydev/gometalinter:latest

