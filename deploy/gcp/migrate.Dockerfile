FROM postgres:17-alpine
COPY db/migrations /migrations
COPY deploy/gcp/migrate.sh /usr/local/bin/epistemic-migrate
RUN chmod +x /usr/local/bin/epistemic-migrate
ENTRYPOINT ["/usr/local/bin/epistemic-migrate"]
