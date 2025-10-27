FROM alpine
COPY ./build_linux/ /app/kx-proxy/

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && apk update && apk add gcompat

EXPOSE 8080
CMD /app/kx-proxy/kx-proxy
