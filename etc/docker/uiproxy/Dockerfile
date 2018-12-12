FROM nginx:1.15-alpine

RUN apk add --no-cache openssl

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

COPY nginx.conf /etc/nginx/nginx.tmpl
RUN rm /etc/nginx/conf.d/default.conf

ENV CLOUDINFO_URL=https://beta.banzaicloud.io/cloudinfo
ENV RECOMMENDER_URL=https://beta.banzaicloud.io/recommender
ENV UI_URL=http://ui/ui

CMD dockerize -template /etc/nginx/nginx.tmpl:/etc/nginx/nginx.conf -stdout /var/log/nginx/access.log -stderr /var/log/nginx/error.log nginx
