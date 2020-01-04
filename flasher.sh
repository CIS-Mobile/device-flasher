#!/bin/bash
#
# Copyright (C) 2019-2020 CIS Mobile
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

while [ "$1" != "" ]; do
    case $1 in
        --factory-image )       shift
                                FACTORY_IMAGE=$1
                                ;;
        --altos-image )         shift
                                ALTOS_IMAGE=$1
                                ;;
        --altos-verity-key )    shift
                                ALTOS_KEY=$1
                                ;;
        --skip-relocking )      SKIP_RELOCKING=true
                                ;;
    esac
    shift
done

if ! grep -q partition-exists $(which fastboot); then
  echo "fastboot too old; please download the latest version at https://developer.android.com/studio/releases/platform-tools.html"
  exit 1
fi

# Create staging directory
tmpdir=$(mktemp -d)

# Extract images into staging directory
unzip -d ${tmpdir}/factory $FACTORY_IMAGE

echo "Enable Developer Options on device (Settings -> About Phone -> tap \"Build number\" 7 times)"
read -p "Press enter to continue"
echo "Enable USB debugging on device (Settings -> System -> Advanced -> Developer Options) and allow the computer to debug (hit \"OK\" on the popup when USB is connected)"
read -p "Press enter to continue"
echo "Enable OEM Unlocking (in the same Developer Options menu)"
read -p "Press enter to continue"

adb reboot bootloader
sleep 5

# Unlock the bootloader
echo "Unlocking bootloader..."
echo "Please use the volume and power keys on the device to confirm."
fastboot --skip-reboot flashing unlock
read -p "Press enter to continue once you have confirmed unlocking on the device"
sleep 5

# Flash factory image on both slots
echo "Ensuring proper firmware is installed..."
fastboot --slot all flash bootloader $(find ${tmpdir}/factory -name bootloader*)
fastboot reboot-bootloader
sleep 5
fastboot --slot all flash radio $(find ${tmpdir}/factory -name radio*)
fastboot reboot-bootloader
sleep 5
fastboot --skip-reboot update $(find ${tmpdir}/factory -name image*)
fastboot reboot-bootloader
sleep 5

# Flash altOS image
echo "Installing altOS..."
fastboot --skip-reboot update $ALTOS_IMAGE
# Flash custom AVB key if passes to script (Some devices don't have this, so only do it on ones that do)
if [ ! -z "$ALTOS_KEY" ]; then
  fastboot flash avb_custom_key $ALTOS_KEY
fi

# Wiping data
echo "Formatting data and prepping device..."
fastboot -w reboot-bootloader
sleep 5

# Re-lock the bootloader
if [ "$SKIP_RELOCKING" != "true" ]; then
  echo "Re-locking bootloader..."
  echo "Please use the volume and power keys on the device to confirm."
  fastboot flashing lock
  read -p "Press enter to continue once you have confirmed locking on the device"
else
  echo "Skipping bootloader relocking..."
fi

# Clean up
echo "Cleaning up..."
rm -rf ${tmpdir}

# Format data and reboot
echo "Done! Rebooting..."
fastboot reboot
