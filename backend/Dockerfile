# --- Compilación ---
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
# Carpeta de adjuntos; la imagen final no tiene shell para crearla.
RUN mkdir -p /out/data/uploads

# --- Imagen final: solo el binario (~15 MB) ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV UPLOAD_DIR=/data/uploads
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
