#pragma once

#include "driver/gpio.h"

/* LILYGO T-A7670G/E/SA R2 board wiring. */
#define BOARD_POWER_EN GPIO_NUM_12
#define MODEM_TX_PIN   GPIO_NUM_26
#define MODEM_RX_PIN   GPIO_NUM_27
#define MODEM_PWRKEY   GPIO_NUM_4
#define MODEM_RESET    GPIO_NUM_5
#define MODEM_DTR      GPIO_NUM_25
