# AluminiumServer/Dockerfile
FROM node:26-alpine

WORKDIR /app

# Copy package files and install dependencies free bird
COPY package*.json ./
RUN npm ci

# Copy the rest of the application source code
COPY . .


EXPOSE 3000

RUN npx tsc

CMD ["node", "dist/server.js"]