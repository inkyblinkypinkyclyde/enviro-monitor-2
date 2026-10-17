# Enviro Monitor

Home temperature and humidity monitoring. Small ESP8266 boards with DHT22 sensors sit in each room and serve their latest reading over HTTP. A Go API polls those boards, stores the readings in Postgres, and a React dashboard shows the current values and 24-hour history per room.

```
ESP8266 + DHT22  <--- GET /status ---  Go API  ---> Postgres
 (one per room)                          ^
                                         |  /api/*
                                    React dashboard
```

## Repository layout

| Path        | Contents                                                      |
| ----------- | ------------------------------------------------------------- |
| `nodes/`    | ESP8266 firmware (Arduino sketch) and flashing Makefile       |
| `backend/`  | Go API, Postgres schema and seed data ([backend/README.md](backend/README.md)) |
| `frontend/` | React + TypeScript + Vite dashboard                           |

## Quick start (server side)

Requires Docker with Compose.

```sh
docker compose up --build
```

- Dashboard: http://localhost:3000
- API: http://localhost:8080 (the dashboard proxies it at `/api`)
- Postgres: `localhost:5432` (user/password `postgres`, database `enviro_monitor`; override with `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`)

The schema is applied on first start of an empty database volume. Use `make rebuild` to rebuild the API and frontend images.

Add a room from the dashboard by giving it a name and the IP address of its node (see below). The API stores readings only when asked to poll the nodes:

```sh
curl -X POST http://localhost:8080/collect
```

Call this on a schedule (for example from cron every few minutes) to build up history.

### Running without Docker

```sh
# Backend (see backend/README.md)
cd backend
make db-setup && make db-seed   # seed is optional
cd api && go run .

# Frontend (proxies /api to http://localhost:8080, override with API_URL)
cd frontend
npm install
npm run dev
```

## Nodes (ESP8266 firmware)

Each node is an ESP8266 reading a DHT22 sensor. The sketch in [nodes/esp8266/esp8266.ino](nodes/esp8266/esp8266.ino) is written for an ESP-01S ("Generic ESP8266 Module"). It joins your Wi-Fi, reads the sensor every 20 seconds and serves:

| Endpoint  | Response                                                          |
| --------- | ----------------------------------------------------------------- |
| `/`       | Small HTML page with live temperature and humidity                |
| `/status` | JSON: `{"hostname": "...", "temperature": 21.3, "humidity": 48.0}` |

The backend polls `/status` on port 80 of every sensor IP stored in the database.

### Hardware

- An ESP8266 board (ESP-01S, or a NodeMCU/Wemos D1 mini; see the pin note below)
- A DHT22 (AM2302) sensor
- A USB-serial adapter for flashing ESP-01/ESP-01S boards (3.3 V logic). NodeMCU and D1 mini boards have USB built in.
- Jumper wires

Wiring for the sketch as written (`DHTPIN 0`, i.e. GPIO0):

| DHT22 pin | ESP8266                    |
| --------- | -------------------------- |
| VCC       | 3.3 V                      |
| DATA      | GPIO0 (with a 4.7-10 kΩ pull-up to 3.3 V) |
| GND       | GND                        |

Notes:

- Power the ESP8266 from 3.3 V only. 5 V will damage it.
- GPIO0 is also the boot-mode pin. It must be pulled low at power-up to enter flash mode, so the DHT22 data line should be disconnected, or at least not driving the pin, while flashing. Reconnect it afterwards.
- If you use a different pin, change `DHTPIN` in the sketch. If you use a DHT11 or DHT21, change `DHTTYPE`.

### 1. Install the toolchain

Install [arduino-cli](https://arduino.github.io/arduino-cli/latest/installation/) (the Makefile calls it, override with `ARDUINO_CLI=...` if it is not on your `PATH`).

```sh
arduino-cli config init
arduino-cli config add board_manager.additional_urls https://arduino.esp8266.com/stable/package_esp8266com_index.json
arduino-cli core update-index
arduino-cli core install esp8266:esp8266
```

Install the libraries the sketch includes:

```sh
arduino-cli lib install "DHT sensor library" "Adafruit Unified Sensor"
```

`ESPAsyncTCP` and `ESPAsyncWebServer` are not in the Arduino library index, so install them from git (enable unsafe installs first):

```sh
arduino-cli config set library.enable_unsafe_install true
arduino-cli lib install --git-url https://github.com/me-no-dev/ESPAsyncTCP
arduino-cli lib install --git-url https://github.com/me-no-dev/ESPAsyncWebServer
```

(`Hash.h` and `ESP8266HTTPClient.h` ship with the ESP8266 core.)

On Linux, make sure your user can access serial devices, then log out and back in:

```sh
sudo usermod -aG dialout $USER   # some distros use the "uucp" group instead
```

### 2. Configure Wi-Fi and the device name

```sh
cd nodes/esp8266
cp config.example.h config.h
```

Edit `config.h`:

```c
#define WIFI_SSID "your-wifi-ssid"
#define WIFI_PASSWORD "your-wifi-password"
#define DEVICE_HOSTNAME "Living room temperature"
```

`config.h` is git-ignored so credentials are not committed. The ESP8266 only supports 2.4 GHz networks. `DEVICE_HOSTNAME` is shown on the node's web page and returned by `/status`.

### 3. Put the board in flash mode

How you do this depends on the board:

- **NodeMCU / Wemos D1 mini:** plug in via USB. Flashing is normally automatic.
- **ESP-01 / ESP-01S with a USB-serial adapter:**
  1. Wire adapter 3.3 V to VCC and CH_PD/EN, GND to GND, adapter TX to ESP RX, adapter RX to ESP TX.
  2. Connect GPIO0 to GND.
  3. Power-cycle the board (unplug and replug the adapter, or briefly pull RST to GND) so it boots into the serial bootloader.
  4. Plug the adapter into your computer.

  Many adapters cannot supply enough current for Wi-Fi. If uploads fail randomly, use a dedicated 3.3 V supply (sharing GND with the adapter).

### 4. Find the serial port

From `nodes/`:

```sh
make ports
```

Look for something like `/dev/ttyUSB0` or `/dev/ttyACM0` (`COM3` on Windows).

### 5. Compile and flash

```sh
cd nodes
make compile                    # check that the sketch builds
make flash PORT=/dev/ttyUSB0    # compile and upload
```

`make upload` is an alias of `make flash`. Defaults can be overridden:

```sh
make flash PORT=/dev/ttyUSB0 FQBN=esp8266:esp8266:nodemcuv2   # e.g. NodeMCU
```

Use `arduino-cli board listall esp8266` to find the FQBN for your board. The default is `esp8266:esp8266:generic`.

### 6. Run it and find its IP address

1. For ESP-01 boards, disconnect GPIO0 from GND, reconnect the DHT22 data line and power-cycle the board so it boots normally.
2. Open the serial monitor at 115200 baud:

   ```sh
   arduino-cli monitor -p /dev/ttyUSB0 -c baudrate=115200
   ```

3. After "Connecting to WiFi" the board prints its IP address. Check it in a browser or with curl:

   ```sh
   curl http://<node-ip>/status
   ```

4. Give the node a fixed address (a DHCP reservation in your router) so the stored IP does not change, then add the room and IP in the dashboard.

### Troubleshooting

| Symptom                                     | Likely cause                                                       |
| ------------------------------------------- | ------------------------------------------------------------------ |
| `Set PORT, for example PORT=/dev/ttyUSB0`   | `make flash` needs the `PORT` argument                             |
| Permission denied on the port               | User not in the `dialout`/`uucp` group                             |
| `Failed to connect to ESP8266` / timeouts   | Board not in flash mode (GPIO0 not low at boot), TX/RX swapped, or weak 3.3 V supply |
| Compile error: `ESPAsyncWebServer.h` missing | Async libraries not installed (step 1)                            |
| `Failed to read from DHT sensor!` in serial | Wiring, missing pull-up, or wrong `DHTTYPE`/`DHTPIN`               |
| Stuck on `Connecting to WiFi`               | Wrong credentials in `config.h` or a 5 GHz-only network            |
| Boots but never shows an IP, or reboots     | GPIO0 still held low, or DHT22 data line pulling the pin low at boot |

## Tests and pre-commit checks

```sh
make -C backend test      # Go unit tests
npm --prefix frontend test
```

Both run automatically on commit through [pre-commit](https://pre-commit.com), along with gitleaks and basic file checks:

```sh
pip install pre-commit
pre-commit install
```
