FROM mkboudreau/go-present:latest

# Update certs
RUN apt-get update && apt-get install -y ca-certificates
COPY ./certs/* /etc/ssl/certs/
RUN update-ca-certificates