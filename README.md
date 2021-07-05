simple-canary
=============

A simple canary monitor.

Watches out for pings from a device/script/server, and if a ping hasn't been received within a predetermined window, marks that device/script/server it as "offline" on a status webpage.

This is useful because it allows for integration of simple devices with uptime monitoring tools like uptimerobot.  These devices can be simple IoT connected devices, cronjobs that need to regularly run, or even servers.

Uptimerobot allows for up to 50 checks on its free service, however their ping service called "Heartbeat" only exists on the paid accounts.  Simple-canary solves this problem.

This tool allows you to use their http/https or keyword monitor methods to have the same capabilities as their "Heartbeat" service but for free.

Usage Scenario
--------------

You have a number of IoT devices, servers and cronjobs that you need to know are working and are online.

1. You configure each server, IoT device and cronjob to "checkin" to a unique simple-canary url with a token (password)
1. Uptimerobot is then configured to monitor a specific "status" page that simple-canary presents for each server, iot device or cronjob
1. Simple-canary watches for the "checkins" from each device, and if one hasn't been received within a specified time limit, it will update the status page that uptimerobot monitors
1. Uptimerobot then sees that a particular device is offline and does whatever actions you have defined

Installation
------------
You can use any of these methods to install:
- `# go get -v github.com/smford/simple-canary`

Clone git repo and build yourself
- `# git clone git@github.com:smford/simple-canary.git`
- `cd simple-canary`
- `# go build`

Via Docker
- something
- more


Configuration
-------------

For simple-canary to work you must configure three things:
- the simple-canary server
- configured each device to checkin to simple-canary
- configure uptimerobot to monitor the devices specific status page

## Configuring the Server
Create a configuration file called `config.yaml` an example is available below:
  ```
  checkintoken: mycheckintoken
  statustoken: mystatustoken
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

### Configuration File Options
| Setting | Details |
|:--|:--|
| checkintoken | Token used by a device to checkin |
| statustoken | Token used to display status information |
| statustokencheck | Use statustoken or not |
| listenip | The IP for simple-canary to listen to, 0.0.0.0 = all IPs |
| listenport | The port that simple-canary should listen on |
| indexhtml | the name and path to the file that is shown when people visit the main page of simple-canary |
| ttl | The number of seconds to wait after a checkin before marking a device as offline|
| devices | A list of devices to accept checkins for |
| verbose | Enable verbose mode.  Note tokens will be displayed in the logs |

Starting simple-canary
----------------------

After configuring the config.yaml in the same directory as the simple-canary executable, simply:

`# simple-canary`


If you want to have the configuration file in a different location, you can start simple-canary like so:

`# simple-canary --config /path/to/config.yaml`

Configure Clients
-----------------

- IoT Device, assuming the IoT device is called "frontdoor"
  Configure it to do an http request to: `http://192.168.10.1:54034/checkin/frontdoor?token=mycheckintoken`
- Cronjob: Add the following line to the end of the script that is run by your cronjob: `wget --spider "http://192.168.10.1:54034/checkin/cronjob1?token=mycheckintoken" >/dev/null 2>&1`
- Server: Add the following cronjob causing the server to checkin ever 5 minutes
`*/5 * * * * wget --spider "http://192.168.10.1:54034/checkin/server1?token=mycheckintoken" >/dev/null 2>&1`


Configuring UptimeRobot.com Monitors
------------------------------------


Command Line Options
--------------------
```
  --config [config file]             Configuration file: /path/to/file.yaml, default = ./config.yaml
  --displayconfig                    Display configuration
  --help                             Display help
  --version                          Display version
```


To Do
-----
- per device TTL
- per device checkintoken
