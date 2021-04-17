build:
	go mod download
	go build

run:
	make
	./bluebao 2> /dev/null

server-debug:
	go build -o tempserver
	./tempserver -readConf=true

client-debug:
	go build -o tempclient
	sleep 1
	./tempclient

watch:
	ls src/server/* src/ui/* src/utils/*  main.go | entr -rc make run

watch-server:
	ls src/server/* src/ui/* src/utils/*  main.go | entr -rc make server-debug

watch-client:
	ls src/client/* src/ui/* src/utils/*  main.go | entr -rc make client-debug
