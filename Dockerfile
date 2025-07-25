FROM public.ecr.aws/lambda/provided:al2 AS builder

RUN yum install -y golang git tar xz
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
RUN go build -tags lambda.norpc -o bootstrap ./cmd/main.go

FROM public.ecr.aws/lambda/provided:al2

COPY layers/ffmpeg-layer/bin/ffmpeg /opt/bin/ffmpeg
COPY layers/ffmpeg-layer/bin/ffprobe /opt/bin/ffprobe
COPY layers/ytdlp-layer/bin/yt-dlp    /opt/bin/yt-dlp
RUN chmod +x /opt/bin/ffmpeg /opt/bin/ffprobe /opt/bin/yt-dlp

COPY --from=builder /src/bootstrap   /var/task/bootstrap
RUN chmod +x /var/task/bootstrap

CMD [ "/bootstrap" ]
