#!/usr/bin/env bash
set -e

# Disallow usages of functions that cause the program to exit in the library code
SCRIPT_PATH=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
EXCLUDE_DIRECTORIES="--exclude-dir=examples --exclude-dir=e2e --exclude-dir=cmd --exclude-dir=.git --exclude-dir=.github --exclude-dir=test"
# TODO(ar): Add back panic
DISALLOWED_FUNCTIONS=('os.Exit(' 'Fatal(' 'Fatalf(' 'Fatalln(' 'fmt.Println(' 'fmt.Printf(' 'log.Print(' 'log.Println(' 'log.Printf(')


for disallowedFunction in "${DISALLOWED_FUNCTIONS[@]}"
do
	if grep -R $EXCLUDE_DIRECTORIES -e "$disallowedFunction" "$SCRIPT_PATH/.." | grep -v -e '_test.go' -e 'nolint'; then
		echo "$disallowedFunction may only be used in example code"
		exit 1
	fi
done
