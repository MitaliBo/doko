FROM nginx

ADD default.conf /etc/nginx/conf.d/default.conf

LABEL docons/name=docons-demo-service
LABEL docons/port=8080
LABEL docons/tags=demo,nginx
LABEL docons/meta-sidecar-mode=http
LABEL docons/meta-sidecar-http-prefix=/hello/world
