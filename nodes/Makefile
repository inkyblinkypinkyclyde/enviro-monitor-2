ARDUINO_CLI ?= arduino-cli
FQBN ?= esp8266:esp8266:generic
SKETCH ?= esp8266
PORT ?=

NODES_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
SKETCH_DIR := $(NODES_DIR)$(SKETCH)

.PHONY: help compile upload flash ports

help:
	@printf '%s\n' \
	  'Targets:' \
	  '  compile          Compile the selected sketch' \
	  '  upload PORT=...  Compile and upload to the selected serial port' \
	  '  ports            List available serial ports' \
	  '' \
	  'Options: FQBN=esp8266:esp8266:generic SKETCH=esp8266'

compile:
	$(ARDUINO_CLI) compile --fqbn "$(FQBN)" "$(SKETCH_DIR)"

upload:
	@test -n "$(PORT)" || { echo 'Set PORT, for example PORT=/dev/ttyUSB0' >&2; exit 2; }
	$(ARDUINO_CLI) compile --fqbn "$(FQBN)" --port "$(PORT)" --upload "$(SKETCH_DIR)"

flash: upload

ports:
	$(ARDUINO_CLI) board list
