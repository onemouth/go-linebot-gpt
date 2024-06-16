.PHONY: clean

bot:
	go build -o bot app/bot/main.go

clean:
	rm -f bot

docker-build: clean
	GOOS=linux GOARCH=amd64 go build -o bot app/bot/main.go
	docker build . -t ltt45/linebot:latest --platform=linux/arm64

docker-build-push: docker-build
	docker push ltt45/linebot:latest
