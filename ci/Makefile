pipeline:
	fly -t ci sp -p stun -c pipeline.yml --load-vars-from ~/.concourse/credentials.yml
resource:
	@echo "forcing dev branch to update on ci"
	@fly -t ci check-resource -r stun/dev
