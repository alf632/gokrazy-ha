FROM debian:bullseye-slim

RUN apt install gnupg2 && apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 7FA3303E &&
    echo 'deb http://archive.raspberrypi.org/debian/ bullseye main' > /etc/apt/sources.list.d/raspi.list

RUN apt update && apt install -y raspi-config

# setup startup script
COPY entrypoint.sh .
CMD ./entrypoint.sh

