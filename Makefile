build::
	go mod download
	go build

run:: build
	./bluebao

server-debug::
	go build -o tempserver
	./tempserver

client-debug::
	go build -o tempclient
	sleep 1

watch::
	ls main.go | entr -rc make run

watch-server::
	ls main.go | entr -rc make server-debug

watch-client::
	ls main.go | entr -rc make client-debug
