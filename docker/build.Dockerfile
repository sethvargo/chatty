FROM scratch
MAINTAINER Seth Vargo <sethvargo@gmail.com>

ADD "./pkg/linux_amd64/chatty" "/"
ENTRYPOINT ["/chatty"]
