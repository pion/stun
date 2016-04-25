test:
	go test -v
bench:
	go test -bench .
fuzz-prepare-msg:
	go-fuzz-build -func FuzzMessage -o stun-msg-fuzz.zip github.com/cydev/stun
fuzz-prepare-typ:
	go-fuzz-build -func FuzzType -o stun-typ-fuzz.zip github.com/cydev/stun
fuzz-msg:
	go-fuzz -bin=./stun-msg-fuzz.zip -workdir=examples/stun-msg
fuzz-typ:
	go-fuzz -bin=./stun-typ-fuzz.zip -workdir=examples/stun-typ
lint:
	@gometalinter -e "AttrType.+gocyclo" -e "_test.go.+gocyclo"
