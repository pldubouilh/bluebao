FLAGS := -ldflags "-s -w" -trimpath

build::
	go mod download
	go build ${FLAGS} -o bluebao

run:: build
	./bluebao

watch::
	ls main.go | entr -rc make run
