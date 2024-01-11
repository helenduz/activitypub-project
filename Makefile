all:
	go build -o ./server ./cmd/server

clean:
	rm -fv server