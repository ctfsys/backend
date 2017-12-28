From alpine:3.7

ENV BINARY=hackersvc

WORKDIR /

EXPOSE 8080 \
       8081 \
       8082 \
       8083

COPY bin/${BINARY} /

RUN chmod +x /${BINARY}
ENTRYPOINT ["/hackersvc"]
