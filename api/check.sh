#!/usr/bin/env bash
api -c $(ls -dm stun*.txt | tr -d ' \n') -except except.txt github.com/pion/stun
