debug_remote:
	go build -gcflags="all=-N -l" -o ${GOPATH}/bin/wechat ./main.go
	dlv --listen=:2345 --headless=true --api-version=2 exec ${GOPATH}/bin/wechat