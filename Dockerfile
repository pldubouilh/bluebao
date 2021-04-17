FROM ubuntu:focal
ARG DEBIAN_FRONTEND=noninteractive

RUN apt -y update && apt -y install gcc libgtk-3-dev libappindicator3-dev golang ca-certificates

COPY . /bluebao

RUN cd /bluebao && go mod download

RUN cd /bluebao && rm bluebao && go build

# sudo docker build -t bluebao-ubuntu .
# sudo docker run --rm --entrypoint /bin/sh bluebao-ubuntu -c "cat /bluebao/bluebao" > bluebao-ubuntu
# chmod +x bluebao-ubuntu
