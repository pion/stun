pipeline:
	fly -t ci sp -p stun -c pipeline.yml --load-vars-from ~/.concourse/credentials.yml
resource:
	fly -t ci check-resource stun/dev
