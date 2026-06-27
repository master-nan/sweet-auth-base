FROM scratch

WORKDIR /app
COPY bin/sweet_admin /app/sweet_admin
COPY bin/migrate /app/migrate
COPY bin/db-preflight /app/db-preflight
COPY bin/container-entrypoint /app/container-entrypoint
COPY bin/healthcheck /app/healthcheck
COPY config-docker.yaml config-pro.yaml casbin_model.conf /app/

ENV APP_ENV=docker
EXPOSE 9009
CMD ["/app/container-entrypoint"]
