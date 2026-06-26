FROM node:22-alpine

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

EXPOSE 5174

CMD ["npm", "run", "dev", "--", "--port", "5174", "--host", "0.0.0.0"]
