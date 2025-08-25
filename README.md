# 🚄 High-Speed Data Processing for Apex Corse

Infrastructure for real-time data processing and transfer for **Apex Corse**'s car development.

- [🚄 High-Speed Data Processing for Apex Corse](#-high-speed-data-processing-for-apex-corse)
  - [Our Case](#our-case)
  - [Transmitter](#transmitter)
  - [Receiver](#receiver)
  - [Architecture](#architecture)

## Our Case

Our setup is very simple: transmit data between the vehicle and the team at the box. We are going to use 3G/4G techlonogy.

## Transmitter
The transmitter is an **ESP32** with a SIM module. It receives data from the CAN bus and store them in the SD. The ESP32 establishes a MQTTS connection with a raspberry (or any other devices, to be decide). This connection is bidirectional and the server is a provider to be decide. In this way we can also sent data to it, like for changing some settings or setting OTP code. 
Furthermore the ESP32 will have a WiFi module. This will let us to retrive data from the vehicl when we are so near to it. The acces point (AP) will offers an HTML page where we can put some parameters. This AP will have some constraints:
- HTTPS connection (CA previously generated)
- AP with WPA3
- MAC filtering
- Maxx client = 1
- Special character filter in input fields
- OTP token with MQTTS
- AP password change periodically (setted via MQTTS)

---
## Receiver
The receiver is a raspberry or a standard PC. It doesn't matter since all the suite will be a Docekr container. Basically it will acts as:
- Theoretical MQTTS client but it will be the brain
- REDIS server (we can chain it with Grafana or custom App)


---

## Architecture

Currently under development, but there are some ideas:

- **Each** operation in the Transmitter is done by aa thread
- **Every** possible cause of crashing must be handled (connection lost, cyber attack, module crash, esp32 crash)
- **Store** the whole state of the Car every TAU
- **Each** section of the Car has a priority
- **The** SD storage is built in a way that the most recent data is quickly retrieved
- **The** whole state of the Car can be sent
- **We** can communicate with the vehicle thanks to MQTTS, sending crucial information or simple acks.
- **The** transmission needs to be reliable 

> [!NOTE]
> The whole state of the Car will be about 20KB. TAU is a time interval that may be static or dynamic, and we are deciding it together with the Electronics Division.
