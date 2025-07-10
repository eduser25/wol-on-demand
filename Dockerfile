FROM golang:latest

WORKDIR /go/src/wol-on-demand
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/wol-on-demand main.go

FROM gcr.io/distroless/base
COPY --from=0 /go/src/wol-on-demand/bin/wol-on-demand /wol-on-demand 

ENTRYPOINT ["/wol-on-demand"]