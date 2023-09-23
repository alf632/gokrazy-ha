FROM debian:bullseye-slim

RUN apt update && apt install -y dbus bluez

RUN useradd -m bluezuser \
 && passwd -d bluezuser

# setup startup script
COPY entrypoint.sh .
COPY ./bluezuser.conf /etc/dbus-1/system.d/
CMD ./entrypoint.sh

