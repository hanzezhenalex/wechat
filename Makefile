binaries: wechat_server

wechat_server:
	go build -o ${GOPATH}/bin/wechat ./main.go

debug_remote:
	go build -gcflags="all=-N -l" -o ${GOPATH}/bin/wechat ./main.go
	dlv --listen=:2345 --headless=true --api-version=2 exec ${GOPATH}/bin/wechat

docker_wechat_server:
	docker build -f ./Dockerfile --target wechat_server -t alex/wechat_server .

docker_compose:
	docker-compose up .