FROM golang

RUN cd /go \
 && go get github.com/mattn/go-sqlite3 \
 && go get github.com/antchfx/xmlquery 

COPY dialog /ebrmain/bin/

WORKDIR /src
