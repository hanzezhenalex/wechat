FROM golang:1.19 as build

WORKDIR /usr/src/app

COPY . .

RUN make binaries

FROM golang:1.19 as wechat_server

COPY --from=build /go/bin/wechat /usr/bin/wechat

ENTRYPOINT ["/usr/bin/wechat"]
