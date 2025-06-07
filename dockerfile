# Stage 1: Build frontend with Vite
FROM node:22.16 AS frontend-build
WORKDIR /app/Frontend
COPY Frontend/package*.json ./
RUN npm install
COPY Frontend/ ./
# frontend builds into /app/Public/
RUN npm run build

# Stage 2: Build PieceMeal chess bot
FROM gcc:13.1 AS piecemeal-build
WORKDIR /app/PieceMeal
COPY PieceMeal/ ./
# create a build folder and build bot executable into it
RUN mkdir -p /app/Build && gcc -O3 -Wall -Wextra -Werror -march=native -lpthread -o /app/Build/piece-meal PieceMealBot.c

# Stage 3: Build Go backend
FROM golang:1.23 AS backend-build
WORKDIR /app/Backend

COPY Backend/ ./
# build depends on go bindings in PieceMeal
COPY PieceMeal/ /app/PieceMeal
ENV CGO_ENABLE=1

# create a build folder and build server executable into it
RUN mkdir -p /app/Build && go build -o /app/Build/server .

# Stage 4: Minimal Runtime Image
FROM debian:bookworm-slim

WORKDIR /app

# copy from temp build folders into app
COPY --from=frontend-build /app/Public/ ./Public/
COPY --from=backend-build /app/Build/server ./Build/server
COPY --from=piecemeal-build /app/Build/piece-meal ./Build/piece-meal

# run command

EXPOSE 8080

CMD ["./Build/server", "./Build/piece-meal"]