FROM ubuntu:latest

RUN apt-get update
RUN apt-get install -y coturn

USER turnserver

ENTRYPOINT ["/usr/bin/turnserver"]