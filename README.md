# ITM Exporter
ITM Exporter is a Prometheus exporter for IBM Tivoli Monitoring v6.3, IBM Application Performance Management v8 (on-prem only) and IBM OMEGAMON.
The exporter uses ITM REST API in order to collect metrics from IBM ITM/APM/OMEGAMON. Note that ITM REST API is [not officially supported](https://developer.ibm.com/answers/questions/358915/is-itm-rest-api-officially-supported-customer-uses/).


## How to use
Download and unpack the ITM Exporter release. Copy the exporter binary and [config file `config.yaml`](config.yaml) to your ITM v6 TEPS server or IBM APM v8 server.

```
itm_exporter export
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
or
```
itm_exporter --help-long
```


## Prometheus configuration

Add the following job to the `scrape_configs` section:
```yaml
scrape_configs:
  - job_name: 'itm-exporter'
    scrape_interval: 60s
    scrape_timeout: 45s
    static_configs:
      - targets: ['<exporter_ip>:8000']
```
It is not recommended to specify `scrape_interval` less than 60s. 

## ITM Exporter configuration

Example config.yaml:
```yaml
itm_server_url: "http://localhost:15200"
itm_server_user: "sysadmin"
itm_server_password: "pass"
connection_timeout: 8
collection_timeout: 40
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
- `itm_server_user` - for example `sysadmin` for ITM v6 or smadmin for APM v8
- `itm_server_password`
- `connection_timeout` - maximum time allowed for a simple http request from `itm_exporter` to ITM CURI API.
- `collection_timeout` - maximum time allowed for collecting the latest snapshot of metric values for a single Attribute Group.


The section `groups:` specifies which ITM/APM metrics should be collected and exposed by the exporter. ITM exporter concurrently collects metrics for every group. 

- `datasets_uri` - it is a part of the API request URL that identifies particular agent type. The exporter helps a bit in the identification of proper `datasets_uri` for the agent type you'd like to collect. Run the following command:
```
  itm_exporter listAgentTypes --temsName=TEMS 
```
where `temsName` is your ITM TEMS label like `TEMS` or `KD8` if you connect to APM v8 server.
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
- `labels` - list of attributes that should be mapped as Prometheus metric labels (typically string attributes that identify source of the metric like `ORIGINNODE`, `CPUID` or `MOUNTPT`). As a rule of thumb you should always use at least ORIGINNODE and all other metrics that are primary keys within the attribute group as labels (`listAttributes` command shows which metrics are primary keys), otherwise you may see error about duplicate metrics in the `/metrics` output.
- `metrics` - numeric metrics names you'd like to collect.

Attribute names (for mapping with both labels and metrics) can be listed with the following command (example for KLZCPU attribute group within Linux OS dataset):
```
itm_exporter listAttributes --attributeGroup=KLZCPU --dataset=/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets
```
Example output:
```
+------------------------------+------------+-------------+
|         DESCRIPTION          | ATTRIBUTES | PRIMARY KEY |
+------------------------------+------------+-------------+
| System Name                  | ORIGINNODE | false       |
| Time Stamp                   | TIMESTAMP  | false       |
| CPU ID                       | CPUID      | true        |
| User CPU (Percent)           | USRCPU     | false       |
| User Nice CPU (Percent)      | USRNCPU    | false       |
| System CPU (Percent)         | SYSCPU     | false       |
| Idle CPU (Percent)           | IDLECPU    | false       |
| Busy CPU (Percent)           | BUSYCPU    | false       |
| I/O Wait (Percent)           | WAITCPU    | false       |
| User to System CPU (Percent) | USRSYSCPU  | false       |
| Steal CPU (Percent)          | STEALCPU   | false       |
| Recording Time               | WRITETIME  | false       |
+------------------------------+------------+-------------+
```
- `managed_system_group` - the name of the managed system group, grouping agents in scope of the collection.

## ITM Exporter CLI options
```
usage: itm_exporter [<flags>] <command> [<args> ...]

ITM exporter for Prometheus.

Flags:
      --help        Show context-sensitive help (also try --help-long and --help-man).
  -s, --apmServerURL=APMSERVERURL
                    HTTP URL of the CURI REST API server.
  -u, --apmServerUser=APMSERVERUSER
                    CURI API user.
  -p, --apmServerPassword=APMSERVERPASSWORD
                    CURI API password.
      --web.listen-address=":8000"
                    The address to listen on for HTTP requests.
  -v, --verboseLog  Verbose logging for export and diagnostic modes.

Commands:
  help [<command>...]
    Show help.


  listAttributes --attributeGroup=ATTRIBUTEGROUP --dataset=DATASET
    List available attributes for the given attribute group.

    -g, --attributeGroup=ATTRIBUTEGROUP
                           Attribute group
    -d, --dataset=DATASET  Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.

  listAttributeGroups --dataset=DATASET [<flags>]
    List available Attribute Groups for the given dataset.

    -d, --dataset=DATASET  Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.
    -l, --long             List Attributes for every Attribute Group in dataset

  listAgentTypes --temsName=TEMSNAME
    Lists datasets (agent types).

    -t, --temsName=TEMSNAME  ITM TEMS label (specify KD8 for APMv8).

  export
    Start itm_exporter in exporter mode.


  test --file=FILE
    Start itm_exporter in diagnostic mode.

    --file=FILE  JSON response
```

## Prometheus Quick Start

If you are not familiar with Prometheus, a good option is to start with full Prometheus/Grafana stack running in Docker Compose.

1. Install Docker and Docker Compose: https://docs.docker.com/compose/install/
2. `git clone https://github.com/vegasbrianc/prometheus`
3. `cd prometheus`
4. `vi prometheus/prometheus.yml` and add `itm-exporter` job as describe above.
5. `docker-compose up -d`
6. Check Prometheus URL via web browser: http://localhost:9090/targets ad make sure that Prometheus server can scrape `itm-exporter`
7. If the job status is `UP`, access Grafana via web browser: http://localhost:9090/ (admin/foobar).
8. [Import](https://grafana.com/docs/grafana/latest/reference/export_import/) both dashboards included in this repo.

## License
 Under [MIT](LICENSE).