pipeline:
	fly -t ci sp -p stun -c pipeline.yml --load-vars-from ~/.concourse/credentials.yml
resource:
	@echo "forcing dev branch to update on ci"
	@fly -t ci check-resource -r stun/dev
docker:
	sudo docker build -t cydev/gometalinter:1.6.2 images/gometalinter
docker-push:
	sudo docker push cydev/gometalinter:1.6.2

