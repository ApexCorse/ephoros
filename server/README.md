# Server of Ephoros architecture

The server is one of the three components of Ephoros and is written in Go. It is responsible for:

- Starting an MQTT broker.
- Saving incoming data from the MQTT network inside the DB.
- Providing an HTTP API for retrieve non real-time data.

## Project structure

- The `cmd` folder contains the 'main' scripts for spinning up both the MQTT client and the HTTP API.
- The `internal` folder contains the actual code that runs.
- The `docker` folder contains Docker and Docker Compose configuration files.

## Insights

The server's purpose is to manage incoming data from the ESP32 inside the car, process it and distribute it to connected users. It should be hosted in a remote machine, either in the cloud or on a proprietary machine. 

The ESP32 collects data from sensors all around the car and, while connected to the MQTT broker, it sends it to a topic which is reserved to the specific sensor. The topic follows this pattern:

```text
{raw,p}/<Section>/<Module>/<Sensor>
```

where "raw" represents data coming from the ESP32 and "p" the data processed by the server and emitted again.

Imagine a sensor named "NTC-1", located in the first module of the battery. The reserved topic will be:

```text
{raw,p}/Battery/Module-1/NTC-1
```

The payload sent on a "raw" topic must be binary, 12 bytes long: the first 8 are reserved to the timestamp (precision: milliseconds), the remaining 4 are for the value (float/int).

Received the data from a particular topic, the server retrieves the data about the specific sensor and adds a record inside the DB with:
- sensor ID;
- sensor value;
- timestamp.

At the same time, the server emits a new MQTT message with the same topic but with "p" instead of "raw". The interested client will subscribe to this topic to receive data in a more understandable way.

## Configuration

In order to start the server, be sure to create both an `.env` file and a `configuration.json` file in the root directory. This will provide essential configuration for the whole server to work.

### `.env`

The `.env` file needs these entries:

- `DB_URL`: the URL of the database to connect to. This has to be in the form of `postgresql://<user>:<password>@<host>:<port>/<database>`.
- `BROKER_URL`: the URL of the EMQX broker to connect to.

> [!NOTE]
> If you want to run tests you need to add the `BASE_DB_URL` entry to the `.env` file with value: `postgresql://<user>:<password>@<host>:<port>/`.

### `configuration.json`

This file contains configuration about the actual sensors in the car. It allows to recognize which sensor a piece data is sent from.

The file has this format:

```json
{
	"sensors": [...] // sensors' configuration
}
```

where each object of the `"sensors"` array must have these fields:

- `name`, the name of the particular sensor.
- `section`, the section of the car it belongs.
- `module`, the module of the `section` it belongs.

> [!CAUTION]
> Every error in configuration files will prevent the server from starting.

## Start

> [!CAUTION]
> To start the server be sure to have both **Docker** and **Docker Compose** installed.

### Development/Production

To start a development instance of the server, execute the following command from the root directory of the server:

```bash
docker compose -f docker/docker-compose.yaml up
```

This will start:

- All the Go services (data processing, data save in the DB, HTTP API).
- The TimescaleDB.
- The EMQX MQTT broker.

### Testing

To launch tests (with live reload too), run the following command:

```bash
docker compose -f docker/docker-compose.test.yaml up
```

This will start:

- The `go test ./...` script.
- A test TimescaleDB instance.

