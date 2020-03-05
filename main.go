package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/forestgiant/sliceutil"
	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/rs/xid"
	"golang.org/x/net/publicsuffix"

	//	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

var (
	up = prometheus.NewDesc(
		"itm_up",
		"itm_exporter successfully connected to the TEP data provider",
		nil, nil,
	)
	invalidChars = regexp.MustCompile("[^a-zA-Z0-9:_]")
)

// ITMCollector struct
type ITMCollector struct {
}

// Describe Implements prometheus.Collector.
func (c ITMCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
}

var (
	c                          Config
	conf                       = c.getConf()
	app                        = kingpin.New("itm_exporter", "ITM exporter for Prometheus.")
	configFile                 = app.Flag("configFile", "ITM exporter configuration file.").Short('c').Default("config.yaml").String()
	itmServer                  = app.Flag("apmServerURL", "HTTP URL of the CURI REST API server.").Short('s').String()
	itmServerUser              = app.Flag("apmServerUser", "CURI API user.").Short('u').String()
	itmServerPassword          = app.Flag("apmServerPassword", "CURI API password.").Short('p').String()
	listenAddress              = app.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":8000").String()
	debug                      = app.Flag("verboseLog", "Verbose logging").Short('v').Bool()
	listAttributes             = app.Command("listAttributes", "List available attributes for the given attribute group.")
	listAttributesGroup        = listAttributes.Flag("attributeGroup", "Attribute group").Short('g').Required().String()
	listAttributesDataset      = listAttributes.Flag("dataset", "Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.").Short('d').Required().String()
	listAttributeGroups        = app.Command("listAttributeGroups", "List available Attribute Groups for the given dataset.")
	listAttributeGroupsDataset = listAttributeGroups.Flag("dataset", "Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.").Short('d').Required().String()
	listAgentTypes             = app.Command("listAgentTypes", "Lists datasets (agent types).")
	listAgentTypesTEMS         = listAgentTypes.Flag("temsName", "ITM TEMS label (specify KD8 for APMv8).").Short('t').Required().String()
	export                     = app.Command("export", "Start itm_exporter in exporter mode.")
	invalidMetricChars         = regexp.MustCompile("[^a-zA-Z0-9_:]")
)

func init() {
	prometheus.MustRegister(version.NewCollector("itm_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()
	collector := &ITMCollector{}
	registry.MustRegister(collector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

// MakeAsyncRequest makes HTTP request to the CURI API. To be used in a go routine within Collector method
func MakeAsyncRequest(urla string, group string, ch chan<- Result) {
	//start := time.Now()
	var (
		itmUser  string
		itmPass  string
		jsession http.Cookie
	)

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Error(err)
	}
	u, err := url.Parse(urla)
	if *itmServerUser != "" {
		itmUser = *itmServerUser
		itmPass = *itmServerPassword
	} else {
		itmUser = conf.ItmServerUser
		itmPass = conf.ItmServerPassword
	}

	req, err := http.NewRequest("GET", urla, nil)
	req.SetBasicAuth(itmUser, itmPass)
	cli := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
	resp, err := cli.Do(req)
	if err != nil {
		log.Errorln(err)
	}
	defer resp.Body.Close()
	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == "JSESSIONID" {
			jsession = http.Cookie{Name: cookie.Name, Value: cookie.Value, HttpOnly: false}
		}
	}
	log.Debug(jsession)
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Errorln(string(b))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
	}
	//secs := time.Since(start).Seconds()
	//fmt.Printf("%.2f elapsed with response length: %d %s", secs, len(body), url)

	res := new(Result)
	res.body = body
	res.group = group

	urlDelete := strings.Replace(urla, "/items", "", -1)
	req, err = http.NewRequest("DELETE", urlDelete, nil)
	req.SetBasicAuth(itmUser, itmPass)
	req.AddCookie(&jsession)

	resp, err = cli.Do(req)
	if err != nil {
		log.Errorln(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Errorln(string(b))
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
	}
	//fmt.Printf("%.2f elapsed with response length: %d %s", secs, len(body), urlDelete)
	ch <- *res
}

// MakeRequest makes sync HTTP request to the CURI API. For use in CLI commands defined in main()
func MakeRequest(url string) ([]byte, int, error) {

	req, err := http.NewRequest("GET", url, nil)
	//fmt.Println(url)
	if *itmServerUser != "" {
		req.SetBasicAuth(*itmServerUser, *itmServerPassword)
	} else {
		req.SetBasicAuth(conf.ItmServerUser, conf.ItmServerPassword)
	}
	cli := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := cli.Do(req)
	if err != nil {
		log.Errorln(err)
		return nil, 0, err
	}
	defer resp.Body.Close()
	//fmt.Printf("resp.StatusCode: %d\n", resp.StatusCode)
	if resp.StatusCode >= 500 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Errorln(string(b))
		return b, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		log.Errorln("Response status code:", resp.StatusCode)
		return nil, resp.StatusCode, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
	}
	return body, resp.StatusCode, nil
}

// Collect method executes concurrent HTTP requests to the ITM CURI API for each defined Attribute Group and generates Prometheus series
func (c ITMCollector) Collect(ch chan<- prometheus.Metric) {

	var itmServerURL string
	var metricGroup string
	var url string
	var statusCode int

	if *itmServer != "" {
		itmServerURL = *itmServer
	} else {
		itmServerURL = conf.ItmServerURL
	}
	start := time.Now()

	_, statusCode, err := MakeRequest(itmServerURL + "/ibm/tivoli/rest/providers")
	if err == nil && statusCode == 200 {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1)

		itm := make(chan Result)
		for _, group := range conf.Groups {
			guid := xid.New()
			//msys is a special group and goes without "MetricGroup." prefix
			if group.Name == "msys" {
				metricGroup = "/msys"
			} else {
				metricGroup = "/MetricGroup." + group.Name
			}
			if group.Name == "KLZNET" {
				url = itmServerURL + "/ibm/tivoli/rest" + group.DatasetsURI +
					metricGroup + "/items?param_SourceToken=" + group.ManagedSystemGroup +
					"&optimize=true&param_refId=" +
					guid.String() + "&properties=all"
			} else {
				url = itmServerURL + "/ibm/tivoli/rest" + group.DatasetsURI +
					metricGroup + "/items?param_SourceToken=" + group.ManagedSystemGroup +
					"&optimize=true&param_refId=" +
					guid.String() + "&properties=" + strings.Join(group.Labels, ",") + "," +
					strings.Join(group.Metrics, ",")
			}

			//fmt.Println(url)
			go MakeAsyncRequest(url, group.Name, itm)
		}

		for range conf.Groups {
			var (
				items    Items
				labels   []string
				metrics  []string
				attGroup string
			)

			result := <-itm
			json.Unmarshal([]byte(result.body), &items)

			for _, g := range conf.Groups {
				if g.Name == result.group {
					labels = g.Labels
					metrics = g.Metrics
					attGroup = g.Name
				}
			}

			for i := 0; i < len(items.Items); i++ {
				labelmap := make(map[string]string)
				for j := 0; j < len(items.Items[i].Properties); j++ {
					for _, label := range labels {
						if label == items.Items[i].Properties[j].ID {
							if items.Items[i].Properties[j].DisplayValue != "" {
								labelmap[strings.ToLower(label)] = items.Items[i].Properties[j].DisplayValue
							} else {
								labelmap[strings.ToLower(label)] = items.Items[i].Properties[j].Value.String()
							}
						}
					}
				}
				for j := 0; j < len(items.Items[i].Properties); j++ {
					if sliceutil.Contains(metrics, items.Items[i].Properties[j].ID) {
						name := strings.ToLower(invalidChars.ReplaceAllLiteralString(attGroup+"_"+items.Items[i].Properties[j].ID, "_"))
						desc := prometheus.NewDesc(name, "ITM metric "+items.Items[i].Properties[j].Label, nil, labelmap)
						value, err := strconv.ParseFloat(strings.Replace(items.Items[i].Properties[j].DisplayValue, ",", ".", -1), 64)
						if err != nil {
							//fmt.Println(err, items.Items[i].Properties[j].ID)
							value, _ = items.Items[i].Properties[j].Value.Float64()
						}
						ch <- prometheus.MustNewConstMetric(
							desc, prometheus.GaugeValue, value)
					}
				}
			}
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("itm_scrape_duration_seconds", "Time ITM attribute group scrape took.", nil, map[string]string{"group": result.group}),
				prometheus.GaugeValue,
				time.Since(start).Seconds())
		}
	} else {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("itm_scrape_duration_seconds_total", "Time ITM scrape took.", nil, nil),
		prometheus.GaugeValue,
		time.Since(start).Seconds())
}

// getConf reads the config yaml file and returns *Config
func (c *Config) getConf() *Config {

	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return c
}

func main() {
	// log.SetFormatter(&log.TextFormatter{
	// 	DisableColors: true,
	// 	FullTimestamp: true,
	// })
	var itmServerURL string

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	kingpin.HelpFlag.Short('h')

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case listAgentTypes.FullCommand():
		var datasource Datasource
		table.SetHeader([]string{"Agent Type", "Dataset URI"})
		if *itmServer != "" {
			itmServerURL = *itmServer
		} else {
			itmServerURL = conf.ItmServerURL
		}
		responseBody, statusCode, err := MakeRequest(itmServerURL + "/ibm/tivoli/rest/providers/itm." + *listAgentTypesTEMS + "/datasources")
		if err == nil && statusCode == 200 {
			json.Unmarshal([]byte(responseBody), &datasource)
			for i := 0; i < len(datasource.Items); i++ {
				table.Append([]string{datasource.Items[i].Label, datasource.Items[i].DatasetsURI})
			}
			table.Render()
		}
		os.Exit(0)
	case listAttributeGroups.FullCommand():
		var dataset Dataset
		table.SetHeader([]string{"Description", "Attribute Group"})
		if *itmServer != "" {
			itmServerURL = *itmServer
		} else {
			itmServerURL = conf.ItmServerURL
		}
		responseBody, statusCode, err := MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributeGroupsDataset)
		if err == nil && statusCode == 200 {
			json.Unmarshal([]byte(responseBody), &dataset)
			for i := 0; i < len(dataset.Items); i++ {
				table.Append([]string{strings.Replace(dataset.Items[i].Label, "\n", "", -1),
					strings.Replace(dataset.Items[i].ID, "MetricGroup.", "", -1)})
			}
			table.Render()
		}
		os.Exit(0)
	case listAttributes.FullCommand():
		var column Columns
		var responseBody []byte
		var err error
		var statusCode int

		table.SetHeader([]string{"Description", "Attributes"})
		if *itmServer != "" {
			itmServerURL = *itmServer
		} else {
			itmServerURL = conf.ItmServerURL
		}
		if *listAttributesGroup == "msys" {
			responseBody, statusCode, err = MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributesDataset + "/msys/columns")
		} else {
			responseBody, statusCode, err = MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributesDataset + "/MetricGroup." + *listAttributesGroup + "/columns")
		}
		if err == nil && statusCode == 200 {
			json.Unmarshal([]byte(responseBody), &column)
			for i := 0; i < len(column.Items); i++ {
				table.Append([]string{strings.Replace(column.Items[i].Label, "\n", "", -1),
					strings.Replace(column.Items[i].ID, "MetricGroup.", "", -1)})
			}
			table.Render()
		}
		os.Exit(0)
	case export.FullCommand():
		log.Info("Starting itm_exporter in export mode...")
		log.Info("Author: Rafal Szypulka")
	}

	c := ITMCollector{}
	prometheus.MustRegister(c)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head>
            <title>ITM Exporter</title>
            </head>
            <body>
            <h1>ITM Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	http.HandleFunc("/metrics", handler)
	log.Info("itm_exporter listening on port:", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
