# Telegram Notify

A simple API server for forwarding message to Telegram accounts

## Usage

### cli

```sh
telegram-notify \
  -l="[ip]:port" \
  -t="<Telegram Token>" \
  -m="<text|html|markdown>" \
  -r="[<from>:<id,...>;]<id,...>"
```

### docker cli

```sh
docker run -d \
  --name=telegram-notify \
  --restart=unless-stopped \
  -p 8000:8000 \
  -e TOKEN="<Telegram Token>" \
  -e MODE="<text|html|markdown>" \
  -e RULE="[<from>:<id,...>;]<id,...>" \
  nfam/telegram-notify
```

### Arguments

| Argument | Environment variable | Default | Function |
| :----:   | ---                  | ---     | ---      |
| `-l`     | `LISTEN`             | `:8000` | IP address (optional) and the port. |
| `-t`     | `TOKEN`              |         | Telegram bot token. |
| `-m`     | `MODE`               | `text`  | Telegram message mode which can be `text`, `html`, or `markdown`. |
| `-r`     | `RULE`               |         | Message forwarding rule(s) in format `"[<from>:<id,...>;]<id,...>"`. See rule rule example. |

### Rule Example 

The rule `service1:11,12;service2:21,22;31,33` indicates:
* Messages from `service1` will be forwarded to accounts `11` and `12`.
* Messages from `service2` will be forwarded to accounts `21` and `22`.
* Messages from the rest will be forwarded to accounts `31` and `32`.

## Forward Message

In order to send a Telegram message, make a POST request to `http://localhost:8000/notify` with 3 optional query `from`, `sound`, and `mode`.
* The body content is the text message,
* `from` query indicates the sender,
* `sound` which can be `on` or `off` (default), indicates whether to ring a tone on the receiver app, and
* `mode` indicates the text mode which is `text`, `html`, or `markdown`. If `mode` is missing, the value from server argument will be employed.

For example:


```sh
curl -X POST \
     -d 'failed to backup' \
     'http://localhost:8000/notify?from=service1&mode=html'
```

will send the following message to accounts `11` and `12`

```html
<b>service1:</b> failed to backup
```