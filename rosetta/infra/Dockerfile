# Build
FROM golang:1.14 AS build

RUN apt update -y && \
    apt install libgmp-dev libssl-dev git -y && \
    apt -y clean all

ENV GOPATH=/root/go
ENV GO111MODULE=on
ENV HMY_PATH=${GOPATH}/src/github.com/nordicenergy
RUN mkdir -p $HMY_PATH

WORKDIR $HMY_PATH

RUN git clone https://github.com/nordicenergy/NordicEnergy.git && \
    git clone https://github.com/nordicenergy/bls.git && \
    git clone https://github.com/nordicenergy/mcl.git

WORKDIR $HMY_PATH/NordicEnergy

RUN make linux_static && \
    cp ./bin/NordicEnergy /root/NordicEnergy && \
    cp ./rosetta/infra/run.sh /root/run.sh

# Execution
FROM ubuntu:latest

RUN apt update -y && \
    apt install libgmp-dev libssl-dev -y && \
    apt -y clean all

WORKDIR /root

COPY --from=build /root/NordicEnergy /root/NordicEnergy
COPY --from=build /root/run.sh /root/run.sh

EXPOSE 9500/tcp
EXPOSE 9700/tcp
EXPOSE 9800/tcp
ENTRYPOINT ["/bin/bash","/root/run.sh"]
