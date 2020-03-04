# ITM Exporter
ITM Exporter is a Prometheus exporter for IBM Tivoli Monitoring v6 and IBM Application Performance Management v8 (on-prem).
The exporter uses ITM REST API to collect metrics from IBM ITM/APM. Note that ITM REST API is [not oficially supported](https://developer.ibm.com/answers/questions/358915/is-itm-rest-api-officially-supported-customer-uses/).


## How to use
Download and unpack the ITM Exporter [release](https://github.com/rafal-szypulka/itm_exporter/releases). Copy the exporter binary and [config file](config.yaml) to your ITM v6 TEPS server or IBM APM v8 server.

```
itm_exporter export
```
If your config file name is different than `config.yaml`

```
itm_exporter export -c <config_file_name>.yaml
```
The above should start an exporter on default port `8000`. You can check if it works using:
```
curl http://localhost:8000/metrics
```
Use flag `--web.listen-address=<port>` to specify port different than `8000`.

Check all available options with:
```
itm_exporter --help
```

## Prometheus configuration

Add the following job to the `scrape_configs` section:
```yaml
scrape_configs:
  - job_name: 'itm-exporter'
    scrape_interval: 60s
    scrape_timeout: 30s
    static_configs:
      - targets: ['<exporter_ip>:8000']
```
It is not recommended to specify `scrape_interval` less than 60s. 

## ITM Exporter configuration

Example config.yaml:
```yaml
itm_server_url: "http://localhost:15210"
itm_server_user: "sysadmin"
itm_server_password: "pass"
groups: 
- name: "KLZCPU"
  datasets_uri: "/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets"
  labels: ["CPUID", "ORIGINNODE"]
  metrics: ["BUSYCPU", "IDLECPU", "SYSCPU", "USRCPU", "WAITCPU"]
  managed_system_group: "*LINUX_SYSTEM"
- name: "KLZVM"
  datasets_uri: "/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets"
  labels: ["ORIGINNODE"]
  metrics: ["MEMUSEDPCT", "VSFREEPCT"]
  managed_system_group: "*LINUX_SYSTEM"
- name: "KLZDISK"
  datasets_uri: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'
  labels: ["ORIGINNODE", "DSKNAME", "MOUNTPT"]
  metrics: ["DSKFREEPCT", "DSKUSEDPCT", "DSKFREE", "DSKUSED", "INDFREEPCT"]
  managed_system_group: "*LINUX_SYSTEM"
```

- `itm_server_url` - HTTP URL of your TEPS or APM Server, ex.: "http://localhost:15210"
- `itm_server_user` - for example `sysadmin` for ITM v6 or `smadmin` for APM v8
- `itm_server_password`

The section `groups:` specifies which ITM/APM metrics should be collected and exposed by the experter. ITM exporter asynchronically collects metrics for every group. 

- `datasets_uri` - it is a part of the API request URL that identifies particular agent type. The exporter can help a bit in the identification of proper `datasets_uri` for the agent type you'd like to collect. Run the following command to list all supported monitoring agent types on your ITM or APM server:
```
  itm_exporter listAgentTypes --temsName=TEMS 
```
where `temsName` is your ITM TEMS label like `TEMS` (or `KD8` if you connect to APM v8 server).
Example output:
```
+---------------------------------+--------------------------------------------------------------------+
|           AGENT TYPE            |                            DATASET URI                             |
+---------------------------------+--------------------------------------------------------------------+
| Tivoli Enterprise Portal Server | /providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC153/datasets |
| Windows OS                      | /providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC021/datasets |
| All Managed Systems             | /providers/itm.TEMS/datasources/TMSAgent.%26IBM.STATIC000/datasets |
| Summarization and Pruning Agent | /providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC066/datasets |
| Warehouse Proxy                 | /providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC122/datasets |
| Linux OS                        | /providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets |
| UISolution.manager              | /providers/itm.TEMS/datasources/UISolution.manager/datasets        |
+---------------------------------+--------------------------------------------------------------------+
```
- `name` - name of the group you'd like to collect. You can list attribute group names with the following command (example for Linux OS dataset):
```
itm_exporter listAttributeGroups --dataset=/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets
```
Example output:
```
+--------------------------------------+-----------------+
|             DESCRIPTION              | ATTRIBUTE GROUP |
+--------------------------------------+-----------------+
| Linux Network                        | KLZNET          |
| Agent Operations Log                 | OPLOG           |
| Linux NFS Statistics (Superseded)    | LNXNFS          |
| Linux Sockets Detail                 | KLZSOCKD        |
| Linux Disk Usage Trends (Superseded) | LNXDU           |
| Linux Group                          | LNXGROUP        |
| Linux Process (Superseded)           | LNXPROC         |
| Linux CPU                            | KLZCPU          |
| Linux IP Address                     | LNXIPADDR       |
| Linux Process                        | KLZPROC         |
| Linux File Comparison                | LNXFILCMP       |
| Linux Sockets Status                 | KLZSOCKS        |
| Linux System Statistics (Superseded) | LNXSYS          |
| Situation Event Information          | events          |
| Linux IO Ext                         | KLZIOEXT        |
| Linux NFS Statistics                 | KLZNFS          |
| Linux CPU Averages (Superseded)      | LNXCPUAVG       |
| CustomScriptsRuntime Sampled         | KLZSCRTSM       |
| Linux OS Config                      | LNXOSCON        |
| Linux Host Availability              | LNXPING         |
| Linux Sockets Detail (Superseded)    | LNXSOCKD        |
| Configuration Information            | KLZPASCAP       |
| Linux VM Stats (Superseded)          | LNXVM           |
| Linux RPC Statistics (Superseded)    | LNXRPC          |
| Linux System Statistics              | KLZSYS          |
| Linux File Information               | LNXFILE         |
| Linux Swap Rate                      | KLZSWPRT        |
| Alerts Table                         | KLZPASALRT      |
| Linux Sockets Status (Superseded)    | LNXSOCKS        |
| Managed System Information           | msys            |
| CustomScriptsRuntime                 | KLZSCRRTM       |
| Linux IO Ext (Superseded)            | LNXIOEXT        |
| Linux TCP Statistics                 | KLZTCP          |
| CustomScripts                        | KLZSCRPTS       |
| Linux Disk IO                        | KLZDSKIO        |
| Agent Availability Management Status | KLZPASMGMT      |
| Linux All Users                      | LNXALLUSR       |
| Linux RPC Statistics                 | KLZRPC          |
| Linux Disk Usage Trends              | KLZDU           |
| Linux User Login                     | KLZLOGIN        |
| Linux CPU Config                     | LNXCPUCON       |
| Linux Disk (Superseded)              | LNXDISK         |
| Linux Machine Information            | LNXMACHIN       |
| Linux Swap Rate (Superseded)         | LNXSWPRT        |
| Linux Disk                           | KLZDISK         |
| Managed System Groups                | mgrp            |
| Linux Process User Info (Superseded) | LNXPUSR         |
| Agent Active Runtime Status          | KLZPASSTAT      |
| Linux Process User Info              | KLZPUSR         |
| Linux CPU Averages                   | KLZCPUAVG       |
| Linux Network (Superseded)           | LNXNET          |
| Linux Disk IO (Superseded)           | LNXDSKIO        |
| Linux LPAR                           | KLZLPAR         |
| Linux VM Stats                       | KLZVM           |
| Linux CPU (Superseded)               | LNXCPU          |
| Linux File Pattern                   | LNXFILPAT       |
| Linux User Login (Superseded)        | LNXLOGIN        |
| Situation Advice                     | advice          |
+--------------------------------------+-----------------+
```
- `labels` - list of attributes that should be mapped as Prometheus metric labels (typically string attributes that identify source of the metric like `ORIGINNODE`, `CPUID` or `MOUNTPT`)
- `metrics` - numeric metrics names you'd like to collect.
Metric names can be listed with the following command (example for KLZCPU attribute group within Linux OS dataset):
```
itm_exporter listAttributes --attributeGroup=KLZCPU --dataset=/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets
```
Example output:
```
+------------------------------+------------+
|         DESCRIPTION          | ATTRIBUTES |
+------------------------------+------------+
| System Name                  | ORIGINNODE |
| Time Stamp                   | TIMESTAMP  |
| CPU ID                       | CPUID      |
| User CPU (Percent)           | USRCPU     |
| User Nice CPU (Percent)      | USRNCPU    |
| System CPU (Percent)         | SYSCPU     |
| Idle CPU (Percent)           | IDLECPU    |
| Busy CPU (Percent)           | BUSYCPU    |
| I/O Wait (Percent)           | WAITCPU    |
| User to System CPU (Percent) | USRSYSCPU  |
| Steal CPU (Percent)          | STEALCPU   |
| Recording Time               | WRITETIME  |
+------------------------------+------------+
```
- `managed_system_group` - the name of the `Managed System Group` grouping agents in scope of the collection.

## ITM Exporter CLI options
```
usage: itm_exporter [<flags>] <command> [<args> ...]

ITM exporter for Prometheus.

Flags:
      --help                     Show context-sensitive help (also try --help-long and --help-man).
  -c, --configFile="config.yaml"  ITM exporter configuration file.
  -s, --apmServerURL=APMSERVERURL
                                 HTTP URL of the CURI REST API server.
  -u, --apmServerUser=APMSERVERUSER
                                 CURI API user.
  -p, --apmServerPassword=APMSERVERPASSWORD
                                 CURI API password.
      --web.listen-address=":8000"
                                 The address to listen on for HTTP requests.
  -v, --verboseLog               Verbose logging

Commands:
  help [<command>...]
    Show help.


  listAttributes --attributeGroup=ATTRIBUTEGROUP --dataset=DATASET
    List available attributes for the given attribute group.

    -g, --attributeGroup=ATTRIBUTEGROUP
                           Attribute group
    -d, --dataset=DATASET  Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent:
                           '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.

  listAttributeGroups --dataset=DATASET
    List available Attribute Groups for the given dataset.

    -d, --dataset=DATASET  Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent:
                           '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.

  listAgentTypes --temsName=TEMSNAME
    Lists datasets (agent types).

    -t, --temsName=TEMSNAME  ITM TEMS label (specify KD8 for APMv8).

  export
    Start itm_exporter in exporter mode.
```

## Prometheus Quick Start

If you are not familiar with Prometheus, a good option is to start with full Prometheus/Grafana stack running in Docker Compose.

1. Install Docker and Docker Compose: https://docs.docker.com/compose/install/
2. `git clone https://github.com/vegasbrianc/prometheus`
3. `cd prometheus`
4. `vi prometheus/prometheus.yml` and add `itm-exporter` job as described above.
5. `docker-compose up -d`
6. Check Prometheus URL via web browser: http://localhost:9090/targets and make sure that Prometheus server can scrape `itm-exporter`
7. If the job status is `UP`, access Grafana via web browser: http://localhost:9090/ (admin/foobar).
8. [Import](https://grafana.com/docs/grafana/latest/reference/export_import/) both dashboards included in this repo.

## License
 Under [MIT](LICENSE).
