#! /bin/bash -eux

cat request_bods.log | while read bod; do
	curl -X POST 'http://localhost:8080/posts' -d "$bod"
done