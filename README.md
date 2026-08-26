# 🚅 High-Speed Data Processing for Apex Corse

Infrastructure for real-time data processing and transfer for **Apex Corse**'s car development.

## Embedded CAN simulation

The embedded target can feed InfluxDB with synthetic data while no CAN bus is
available. In `menuconfig`, disable **Ephoros CAN receiver → Enable CAN
receiver** and enable **Simulate CAN telemetry**. The build reads the ignored
repository-root `config.dbc`, invokes the existing Vera-based simulator to
export the MQTT topic, name, unit, and bounds for each signal, then embeds that
catalog in the firmware.

Each interval the firmware chooses one DBC signal that has an MQTT topic. It
uses the signal name and unit from the DBC, then draws a Gaussian value centered
on the midpoint of the configured minimum and maximum. The range spans six
standard deviations, and out-of-range samples are rejected. InfluxDB points use
the measurement `can_signal` and retain the DBC MQTT topic as a tag.

The simulator option deliberately fails the build if `config.dbc` or the Go
toolchain is unavailable; normal CAN builds do not require either.

## Private ESP32 configuration

The ESP32 PlatformIO environment reads and updates the ignored
`embedded/sdkconfig.esp32dev.private` file. Bootstrap it once from the tracked
baseline, then use PlatformIO normally:

```sh
cd embedded
cp sdkconfig.esp32dev sdkconfig.esp32dev.private
pio run -t menuconfig
pio run
```

`menuconfig` now saves only the private file, and every `pio run` build uses
that same file. To return to the committed baseline, delete the private file
and copy it again from `sdkconfig.esp32dev`.
