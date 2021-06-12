# Actuator

The actuator is a simple library and tool which allows the Prometheus
Alertmanager to trigger actions via webhooks.

## Usage

To configure the `actuator` server, create a YAML file. By default, `actuator`
looks for `/etc/actuator/actuator.yml`.

**It is your responsibility to make sure you don't put some potentially
catastrophic command behind this webhook!**

```yaml
---
handlers:
    - path: /reboot-the-modem
      token: ThisIsSomeToken
      action: reboot-the-modem
actions:
    - name: reboot-the-modem
      command: |
        ssh root@the-modem "shutdown -r now"
```

Then start the `actuator` server. By default, it listens on `0.0.0.0:9942`. The
full command options are:

```console
$ actuator --help
NAME:
   actuator - Execute commands in response to fired alerts

USAGE:
   actuator [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --server.address value, -a value  ip:port from which to serve Prometheus metrics (default: "0.0.0.0:9942")
   --config.file value, -c value     actuator config file location (default: "/etc/actuator/actuator.yml")
   --help, -h                        show help (default: false)
```

## Alertmanager

Then, configure the Alertmanager to route to the configured webhook. For
example, alertmanager routes using the actuator might look like:

```yaml
route:
  group_by: [alertname, severity, site]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

  receiver: your-default-receiver

  routes:
    # ExternalConnectivityHosed goes to the actuator.
    # This route repeats hourly, which means we reboot the modem hourly while the
    # connection is down.
    - match:
        severity: critical
        alertname: ExternalConnectivityHosed
      receiver: reboot-the-modem
      repeat_interval: 1h
      continue: true
    - match:
        severity: critical
      receiver: pager
```

And the corresponding receivers might look like:

```yaml
receivers:
  - name: your-default-receiver
  - name: pager
    email_configs:
      - to: alerts@example.com
    pagerduty_configs:
      - service_key: youknowwhatevergoeshere
  - name: reboot-the-modem
    webhook_configs:
      - url: "http://localhost:9942/reboot-the-modem"
        http_config:
          authorization:
            credentials: ThisIsSomeToken
```