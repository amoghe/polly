FROM phusion/baseimage:0.9.18

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update && apt-get install --yes --no-install-recommends \
  openjdk-7-jdk \
  emacs24-nox \
  git \
  wget

# No need to clean since baseimage has apt config to clean after every install
