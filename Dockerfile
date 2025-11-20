# Stage 1: Build
# Gunakan versi Go yang stabil (misal 1.21 atau 1.22)
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency & download
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build Binary (Versi Optimized)
# 1. CGO_ENABLED=0: Agar jalan mulus di Alpine
# 2. -ldflags="-s -w": Agar ukuran file kecil (hapus debug info)
# 3. -o main cmd/api/main.go: Lokasi file main kita
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main cmd/api/main.go

# Stage 2: Run (Image Akhir)
FROM alpine:latest

WORKDIR /root/

# Install sertifikat SSL (Penting buat request HTTPS ke Midtrans/Maps)
RUN apk --no-cache add ca-certificates

# Copy binary dari stage builder
COPY --from=builder /app/main .

# Copy file .env (Opsional, biasanya di-inject via docker-compose, tapi gapapa di-copy buat jaga-jaga)
COPY .env .

# Buat folder jika nanti simpan foto lokal (Opsional)
# RUN mkdir uploads

EXPOSE 8080

CMD ["./main"]