FROM golang:1.16
WORKDIR /app
COPY . .
RUN go install
CMD ["go", "run", "main.go"]
