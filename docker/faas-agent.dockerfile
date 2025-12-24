FROM gcr.io/distroless/base-debian12:nonroot

COPY build/faas-agent /faas-agent

EXPOSE 8080

ENTRYPOINT ["/faas-agent"]
CMD ["--config", "/etc/faas/config.yaml", "--env", "dev"]