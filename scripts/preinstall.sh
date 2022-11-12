#! /usr/bin/env sh
groupadd -r thecy
useradd -g thecy -Mr -s /bin/false thecywebsite
mkdir -p /src/thecy/website
