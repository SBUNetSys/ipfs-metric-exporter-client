# This results in a single layer image
FROM python:3.8-alpine

RUN apk add --no-cache git gcc musl-dev bash libc6-compat gcompat

WORKDIR /exporter-server/

COPY . /exporter-server/

ENTRYPOINT ["sh", "./init.sh"]





