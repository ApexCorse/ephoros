# 🚅 High-Speed Data Processing for Apex Corse

Infrastructure for real-time data processing and transfer for **Apex Corse**'s car development.

## Embedded CAN simulation

The embedded target can feed InfluxDB with synthetic data while no CAN bus is
available. In `menuconfig`, disable **Ephoros CAN receiver → Enable CAN
receiver** and enable **Simulate CAN telemetry**. The build reads the ignored
repository-root `config.dbc`, invokes the existing Vera-based simulator to
export its MQTT topic list, and embeds that catalog in the firmware.

Each interval the firmware chooses one DBC MQTT topic and writes a random
numeric `simulated` signal (unit `V`). InfluxDB points use the measurement
`can_signal` and retain the `topic`, `name`, and `unit` tags, so the stored
topic matches the MQTT dashboard hierarchy.

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
