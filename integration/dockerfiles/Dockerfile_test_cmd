FROM gcr.io/distroless/base@sha256:628939ac8bf3f49571d05c6c76b8688cb4a851af6c7088e599388259875bde20 AS first
CMD ["mycmd"]

FROM first
ENTRYPOINT ["myentrypoint"] # This should clear out CMD in the config metadata
