FROM nginx:alpine
LABEL authors="tianlin0"

WORKDIR /app

COPY ./ /app/dist/

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
