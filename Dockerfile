############################
# STEP 1 build executable binary
############################
FROM golang as builder


WORKDIR $GOPATH/salt-api-oauth2-middleware/
COPY . .

RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/salt-api-oauth2-middleware -mod vendor main.go


############################
# STEP 2 ca-certificates
############################
FROM alpine:3 as alpine

RUN apk add -U --no-cache ca-certificates


############################
# STEP 3 build a small image
############################
FROM scratch

ENV GIN_MODE=release
WORKDIR /app/
# Import from builder.
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/salt-api-oauth2-middleware /app/salt-api-oauth2-middleware
ENTRYPOINT ["/app/salt-api-oauth2-middleware"]
EXPOSE 8080