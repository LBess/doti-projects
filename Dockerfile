# Stages:
#   server  the Go API only (what docker-compose uses; run the frontend with `npm run dev`)
#   app     the Go API plus the built frontend, served from one origin (the default; deploy this)

FROM node:22-alpine AS web
WORKDIR /web
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts ./
COPY public ./public
COPY src ./src
# Vite inlines these at build time.
ARG VITE_GOOGLE_MAPS_API_KEY
ARG VITE_GOOGLE_MAP_ID
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server ./
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS server
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]

FROM server AS app
COPY --from=web /web/dist /dist
ENV STATIC_DIR=/dist
