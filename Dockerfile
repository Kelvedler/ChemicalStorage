FROM golang:1.21.4-bookworm

WORKDIR /app

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

COPY . ./

RUN chmod +x docker-entrypoint.sh update.sh

RUN go mod download

RUN go build ./cmd/db/create_admin.go

RUN go build ./cmd/app/chemical_storage.go

EXPOSE 8000

ENTRYPOINT ./docker-entrypoint.sh

