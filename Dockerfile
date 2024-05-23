FROM golang:1.20.14-bullseye as building-stage

# Set the working directory
WORKDIR /go/src

# Copy go.mod and go.sum files first to leverage Docker cache
COPY ./go.mod ./go.sum ./

# Download dependencies based on go.mod and go.sum
RUN go mod download

# Copy the rest of the source code
COPY ./*.go ./
COPY ./Makefile ./

# Build the project using the Makefile
RUN make

FROM fluent/fluent-bit:3.0.4

LABEL maintainer="Björn Franke"

# Copy the built plugin from the building stage
COPY --from=building-stage /go/src/out_rabbitmq.so /fluent-bit/bin/

# Copy the Fluent Bit configuration file
COPY ./conf/fluent-bit-docker.conf /fluent-bit/etc

# Expose the necessary port
EXPOSE 2020

# Set the command to run Fluent Bit with the specified configuration
CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit-docker.conf", "-e", "/fluent-bit/bin/out_rabbitmq.so"]
