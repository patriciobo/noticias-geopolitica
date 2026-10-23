# Imagen de cmd/api para Coolify (ver DEPLOY.md, "API en Coolify").
# El contexto de build es la raíz del repo, no core/: la imagen lleva
# reports/ y config/sources.yaml adentro, igual que el deploy de Render,
# así que cada edición nueva llega con un redeploy (lo dispara daily.yml).

FROM golang:1.26.10-alpine AS build
WORKDIR /src/core
COPY core/go.mod core/go.sum ./
RUN go mod download
COPY core/ ./
# pgx es Go puro: binario estático, sin libc en la imagen final.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# De cada FECHA.audit.json el API solo lee el resumen (model.Provenance);
# la lista de titulares y el estado por medio son ~95% del archivo y solo
# se consultan en el repo ("Registro completo" en la web). Se sacan acá
# para que la imagen no crezca ~300 KB por edición; el repo y la firma
# Sigstore no cambian.
FROM alpine:3.22 AS reports
RUN apk add --no-cache jq
COPY reports/ /reports/
RUN for f in /reports/*.audit.json; do \
      jq -c 'del(.headlines, .sources)' "$f" > "$f.tmp" && mv "$f.tmp" "$f"; \
    done

# Alpine y no distroless: el health check de Coolify corre wget/curl
# adentro del contenedor.
FROM alpine:3.22
RUN adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=build /out/api ./api
COPY config/sources.yaml ./config/sources.yaml
COPY --from=reports /reports/ ./reports/
ENV API_ADDR=":8080" \
    OUT_DIR="/app/reports" \
    SOURCES_PATH="/app/config/sources.yaml"
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["./api"]
