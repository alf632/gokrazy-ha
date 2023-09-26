FROM debian:bullseye-slim

RUN apt update && apt install -y dbus bluez

RUN useradd -m bluezuser \
 && passwd -d bluezuser

# setup startup script
ADD https://raw.githubusercontent.com/alf632/gokrazy-ha/main/bluetooth/entrypoint.sh .
ADD https://raw.githubusercontent.com/alf632/gokrazy-ha/main/bluetooth/bluezuser.conf /etc/dbus-1/system.d/
CMD ./entrypoint.sh

