#!/usr/bin/env bash
api -c $(ls -dm stun*.txt | tr -d ' ') -except except.txt github.com/gortc/stun
