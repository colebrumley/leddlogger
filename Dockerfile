FROM alpine
COPY leddlogger_binary /usr/bin/leddlogger
RUN chmod a+x /usr/bin/leddlogger && apk add --update ca-certificates openssl
ENTRYPOINT /usr/bin/leddlogger
