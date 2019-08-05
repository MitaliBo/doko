FROM nginx

ADD default.conf /etc/nginx/conf.d/default.conf
RUN mkdir -p /var/www/public && echo OK > /var/www/public/_health && echo world > /var/www/public/hello.txt

LABEL doko.name=doko-demo-service
LABEL doko.port=8081
LABEL doko.check=http
LABEL doko.tags=demo,nginx
LABEL doko.meta.key1=val1
