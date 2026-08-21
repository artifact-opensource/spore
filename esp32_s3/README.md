# Spore ESP32-S3 Companion

This directory contains an ESP32-S3 companion scaffold for Spore.

## Included
- Native C firmware entrypoint for ESP-IDF
- GPIO48 RGB activity mapping with Lua-backed profile loading when Lua headers are available
- Command-centre status snapshot hooks
- ESP-NOW/bootstrap stubs
- Shared-firmware partition layout for secondary payloads such as MicroPython bundles
- Host simulation path for validating RGB and profile behavior without hardware

## Layout
- `main/spore_esp32_s3.c` — companion firmware + host simulator
- `main/rgb_profiles.lua` — RGB activity profiles and patterns
- `partitions.csv` — primary app + `spore_assets` partition for staged firmware/assets

## Host simulation
```bash
gcc -DSPORE_HOST_SIM main/spore_esp32_s3.c -o /tmp/spore_esp32_s3_sim $(pkg-config --cflags --libs lua5.4)
/tmp/spore_esp32_s3_sim main/rgb_profiles.lua
```

## Notes
- The native firmware code is written to compile in ESP-IDF when the standard ESP headers are available.
- Lua integration is optional at compile time; if Lua is unavailable the firmware falls back to the built-in `spore-default` profile.
- Secondary firmware and MicroPython payloads are expected to be staged through Spore's shared directory and copied into the `spore_assets` partition during device-specific provisioning.
