# simple-canary

A simple canary checker.

Watches out for pings to a particular url, and if a ping hasn't been received within a certain window flags it as down on a webpage.

This is useful because it allows for integration of simple devices with uptime monitoring tools like uptimerobot.  These devices can be simple IoT connected devices, cronjobs that need to regularly run, or even servers.

Uptimerobot allows for up to 50 checks on its free service, however their ping service called "Heartbeat" only exists on the paid accounts.

This tool allows you to use their http/https or keyword monitor method to have the same capabilities as their "Heartbeat" service.

## Example Usage

Scenario:
- you have 10 iot devices
- you have 5 cron jobs
- you have 20 servers

Each iot device and cronjob is configured to "checkin" to a unique url presented by simple-canary

Uptimerobot is then configured to monitor a specific "status" page for each iot device or cronjob.

Simple-canary watches for the "checkins" from each device, and if one hasn't been received within a specified time limit, it will update the status page that uptimerobot monitors.

Uptimerobot then sees that a particular device is offline and does whatever actions you have defined.


## Configuration

### Configure Server
Create a configuration file called `config.yaml` and example is available below:

```
checkintoken: sometoken
statustoken: statustoken
statustokencheck: false
listenip: 0.0.0.0
listenport: 54034
indexhtml: index.html
ttl: 300
devices:
- frontdoor
- kitchen
- rollerdoor
- laser
- cronjob1
- cronjob2
- server1
verbose: false
```

Run:
`simple-canary --config /path/to/config.yaml`

### Configure Clients

- IoT Device, assuming the IoT device is called "frontdoor"
  Configure it to do an http request to: `http://0.0.0.0:54034/checkin/frontdoor?token=sometoken`
- Cronjob: Add the following line to the end of the script that is run by your cronjob: `wget --spider "http://0.0.0.0:54034/checkin/server1?token=sometoken" >/dev/null 2>&1`
- Server: Add the following cronjob causing the server to checkin ever 5 minutes
`*/5 * * * * wget --spider "http://0.0.0.0:54034/checkin/server1?token=sometoken" >/dev/null 2>&1`


### Configuring UptimeRobot.com Monitors

## Configuration Options
