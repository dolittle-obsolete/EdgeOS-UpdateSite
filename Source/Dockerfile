FROM golang:1.12 as build

COPY Source /build/Source
WORKDIR /build/Source
RUN CGO_ENABLED=0 go build -a -o ../server

FROM scratch
COPY --from=build /build/server /bin/server
ENTRYPOINT [ "/bin/server" ]
EXPOSE 80/tcp 9700/tcp