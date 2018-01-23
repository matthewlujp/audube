FROM matthewishige0528/golang-ffmpeg:v1.0
MAINTAINER Matthew Lu <matthewlujp@gmail.com>

ENV TABLE_NAME "audio_info"

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh
RUN go get -u github.com/golang/dep/cmd/dep

RUN mkdir /app
WORKDIR $GOPATH/src/app
COPY ./*.go ./
COPY ./Gopkg.lock ./
COPY ./Gopkg.toml ./

RUN dep ensure
RUN go build -o /app/main

EXPOSE 1234
CMD ["/bin/sh", "-c", "/app/main > /var/log/server 2>&1"]
