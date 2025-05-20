FROM golang:1.24-alpine

RUN apk add --no-cache \
    git ca-certificates \
    python3 py3-pip \
    build-base libffi-dev openssl-dev

WORKDIR /app

COPY . .

RUN go mod download && go mod verify

RUN python3 -m venv /app/venv \
 && . /app/venv/bin/activate \
 && pip install --no-cache-dir -r requirements.txt

WORKDIR /app/go_code/cmd/
RUN go build -o /app/parky_bot

VOLUME ["/app/session"]

ENTRYPOINT ["/app/parky_bot"]
