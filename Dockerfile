FROM node:lts-alpine as build

WORKDIR /app

COPY package*.json ./

# Install pnpm
RUN npm install -g pnpm

# Install dependencies
RUN pnpm install

# Copy source code to container image
COPY . .

# Build the app
RUN npm run build

# Use the Caddy image
FROM caddy

# Create and change to the app directory
WORKDIR /app

# Copy Caddyfile to the container image
COPY Caddyfile ./

# Copy local code to the container image
RUN caddy fmt Caddyfile --overwrite

# Copy files to the container image
COPY --from=build /app/dist ./dist

# Use Caddy to run/serve the app
CMD ["caddy", "run", "--config", "Caddyfile", "--adapter", "caddyfile"]
