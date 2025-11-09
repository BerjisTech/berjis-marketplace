# Build stage
FROM node:20-alpine AS build
WORKDIR /workspace
COPY clients/angular-auth ./clients/angular-auth
WORKDIR /workspace/marketplace/frontend
COPY marketplace/frontend/package.json ./package.json
COPY marketplace/frontend/yarn.lock ./yarn.lock
RUN corepack enable \
 && if [ -f yarn.lock ]; then yarn install --immutable; else yarn install; fi \
 && ln -s /workspace/marketplace/frontend/node_modules /workspace/node_modules
COPY marketplace/frontend/ .
RUN yarn build

# Runtime stage
FROM nginx:1.27-alpine
# Angular build output path (Angular 18 + browser builder)
COPY --from=build /workspace/marketplace/frontend/dist/marketplace/browser /usr/share/nginx/html
COPY marketplace/frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
