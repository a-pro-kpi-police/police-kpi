FROM golang:1.24 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/kpi-backend .

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=build /out/kpi-backend /app/kpi-backend
COPY kpi_catalog.csv /app/kpi_catalog.csv
COPY import_data /app/import_data

EXPOSE 8000
CMD ["/app/kpi-backend"]

