FROM debian:stable-slim

RUN mkdir /app && mkdir /app/static

RUN apt update && apt install -y ca-certificates

COPY bot /app

WORKDIR /app

ENV OPENAI_API_KEY=

ENV LINE_CHANNEL_SECRET=

ENV LINE_CHANNEL_TOKEN=

ENV PORT=3000

CMD ["./bot"]