#!/bin/bash

# enable I2C and 1-Wire
echo -e "Enabling I2C and 1-Wire Bus"
raspi-config nonint do_i2c 0
