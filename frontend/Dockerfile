FROM scratch

WORKDIR /app
COPY frontend/bin/static-server /app/static-server
COPY frontend/bin/healthcheck /app/healthcheck
COPY frontend/dist/spa /app/public

EXPOSE 80
CMD ["/app/static-server"]
