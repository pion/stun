#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT

api -c $(ls -dm stun*.txt | tr -d ' \n') -except except.txt github.com/pion/stun
