FROM golang:1.24 AS builder
ARG BIU_GITLAB
ARG GIT_USER
ARG GIT_TOKEN
ARG WORKDIR
ARG SERVICE_NAME

WORKDIR ${WORKDIR}
#设置GOPRIVATE, 在go mod操作前生效
ENV GOPRIVATE=${BIU_GITLAB}
ENV GOPROXY="https://goproxy.cn,direct"
ENV GOSUMDB="off"

RUN echo "machine ${BIU_GITLAB}" > ~/.netrc && echo "login ${GIT_USER}" >> ~/.netrc && echo "password ${GIT_TOKEN}" >> ~/.netrc

#优化顺序：优先处理依赖缓存
COPY go.mod go.sum ./
RUN GODEBUG=http2client=0 go mod download -x

RUN go env

#复制应用代码并编译，同时关闭cgo
COPY .. .
RUN CGO_ENABLED=0 go build -gcflags=all="-N -l" -ldflags="-s -w" -o ${SERVICE_NAME} .

# stage2: 运行环境(采用小包alpine基础镜像)
FROM alpine:3.21.3

#ARG CONF_FILE
ARG WORKDIR
ARG GIT_COMMIT_ID
ARG SERVICE_NAME


ENV GIT_COMMIT_ID=${GIT_COMMIT_ID}

WORKDIR ${WORKDIR}
COPY --from=builder ${WORKDIR}/${SERVICE_NAME} .

RUN echo $GIT_COMMIT_ID > ${WORKDIR}/git_commit_id
#RUN mkdir -p etc/language

# 根据参数值复制不同的配置文件
COPY --from=builder ${WORKDIR}/etc/ ./

CMD ["./${SERVICE_NAME}"]