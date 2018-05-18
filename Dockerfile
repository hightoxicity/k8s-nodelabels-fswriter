FROM golang:1.10.2
RUN mkdir -p /go/src/github.com/hightoxicity/k8s-nodelabels-fswriter
WORKDIR /go/src/github.com/hightoxicity/k8s-nodelabels-fswriter
COPY . ./
RUN ls -al /go/src/github.com/hightoxicity/k8s-nodelabels-fswriter
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -v -extldflags -static" -a main.go
ENV ROOTFS /EXTRAROOTFS
RUN mkdir -p ${ROOTFS}

FROM scratch
COPY --from=0 /go/src/github.com/hightoxicity/k8s-nodelabels-fswriter/main /k8s-nodelabels-fswriter
CMD ["/k8s-nodelabels-fswriter"]
