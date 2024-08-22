module github.com/pion/stun

go 1.12

retract [v1.0.0, v1.23.0] // invalid module path

require (
	github.com/pion/dtls/v2 v2.2.7
	github.com/pion/logging v0.2.2
	github.com/pion/transport/v2 v2.2.1
	github.com/stretchr/testify v1.8.4
)
