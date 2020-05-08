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
	"github.com/go-playground/validator"
	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/publicsuffix"
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
	diag         bool
	validate     *validator.Validate
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
	itmServer                  = app.Flag("apmServerURL", "HTTP URL of the CURI REST API server.").Short('s').String()
	itmServerUser              = app.Flag("apmServerUser", "CURI API user.").Short('u').String()
	itmServerPassword          = app.Flag("apmServerPassword", "CURI API password. For export mode you have to specify it in the config file.").Short('p').String()
	listenAddress              = app.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":8000").String()
	verbose                    = app.Flag("verboseLog", "Verbose logging for export and diagnostic modes.").Short('v').Bool()
	listAttributes             = app.Command("listAttributes", "List available attributes for the given attribute group.")
	listAttributesGroup        = listAttributes.Flag("attributeGroup", "Attribute group").Short('g').Required().String()
	listAttributesDataset      = listAttributes.Flag("dataset", "Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.").Short('d').Required().String()
	listAttributeGroups        = app.Command("listAttributeGroups", "List available Attribute Groups for the given dataset.")
	listAttributeGroupsDataset = listAttributeGroups.Flag("dataset", "Dataset (Agent type) URI. You can find it using command: 'itm_exporter listAgentTypes'. Example Dataset URI for Linux OS Agent: '/providers/itm.TEMS/datasources/TMSAgent.%25IBM.STATIC134/datasets'.").Short('d').Required().String()
	listAGLong                 = listAttributeGroups.Flag("long", "List Attributes for every Attribute Group in dataset").Short('l').Bool()
	listAgentTypes             = app.Command("listAgentTypes", "Lists datasets (agent types).")
	listAgentTypesTEMS         = listAgentTypes.Flag("temsName", "ITM TEMS label (specify KD8 for APMv8).").Short('t').Required().String()
	export                     = app.Command("export", "Start itm_exporter in exporter mode.")
	test                       = app.Command("test", "Start itm_exporter in diagnostic mode.")
	testFile                   = test.Flag("file", "JSON response").Required().ExistingFile()
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
	var (
		itmUser  string
		itmPass  string
		jsession http.Cookie
		timeout  time.Duration
	)

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Error(err)
	}
	u, err := url.Parse(urla)
	req, err := http.NewRequest("GET", urla, nil)

	req.SetBasicAuth(conf.ItmServerUser, conf.ItmServerPassword)

	if conf.CollectionTimeout != 0 {
		timeout = conf.CollectionTimeout
	} else {
		timeout = 40
	}

	cli := &http.Client{
		Timeout: timeout * time.Second,
		Jar:     jar,
	}
	resp, err := cli.Do(req)
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()
	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == "JSESSIONID" {
			jsession = http.Cookie{Name: cookie.Name, Value: cookie.Value, HttpOnly: false}
		}
	}

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Error(string(b))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}

	res := new(Result)
	res.body = body
	res.group = group

	urlDelete := strings.Replace(urla, "/items", "", -1)
	req, err = http.NewRequest("DELETE", urlDelete, nil)
	req.SetBasicAuth(itmUser, itmPass)
	req.AddCookie(&jsession)

	resp, err = cli.Do(req)
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Error(string(b))
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}

	ch <- *res
}

// Diag is used only in diagnostic mode (instead of real API request)
func Diag(group string, ch chan<- Result) {
	res := new(Result)
	res.group = group
	ch <- *res
}

// MakeRequest makes sync HTTP request to the CURI API. For use in CLI commands defined in main()
func MakeRequest(url string) ([]byte, int, error) {
	var timeout time.Duration

	req, err := http.NewRequest("GET", url, nil)

	if *itmServerUser != "" {
		req.SetBasicAuth(*itmServerUser, *itmServerPassword)
	} else {
		req.SetBasicAuth(conf.ItmServerUser, conf.ItmServerPassword)
	}
	if conf.ConnectionTimeout != 0 {
		timeout = conf.ConnectionTimeout
	} else {
		timeout = 8
	}
	cli := &http.Client{
		Timeout: timeout * time.Second,
	}
	resp, err := cli.Do(req)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Error(string(b))
		return b, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		log.Error("Response status code:", resp.StatusCode)
		return nil, resp.StatusCode, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}
	return body, resp.StatusCode, nil
}

// Collect method executes concurrent HTTP requests to the ITM CURI API for each defined Attribute Group and generates Prometheus series
func (c ITMCollector) Collect(ch chan<- prometheus.Metric) {

	var (
		itmServerURL string
		metricGroup  string
		url          string
		statusCode   int
	)

	if *itmServer != "" {
		itmServerURL = *itmServer
	} else {
		itmServerURL = conf.ItmServerURL
	}

	start := time.Now()

	_, statusCode, err := MakeRequest(itmServerURL + "/ibm/tivoli/rest/providers")
	if (err == nil && statusCode == 200) || diag {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1)
		itm := make(chan Result)
		for _, group := range conf.Groups {
			guid := xid.New()
			err := validate.Struct(group)
			if err != nil {
				log.Error(err)
			}
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
			log.Debug("ITM API REQUEST: " + url)
			if diag {
				go Diag(group.Name, itm)
			} else {
				go MakeAsyncRequest(url, group.Name, itm)
			}
		}

		for range conf.Groups {
			var (
				items    Items
				labels   []string
				metrics  []string
				attGroup string
			)

			result := <-itm

			if diag {
				byteFile, _ := ioutil.ReadFile(*testFile)
				json.Unmarshal(byteFile, &items)
			} else {
				json.Unmarshal([]byte(result.body), &items)
			}

			//log.Debug(string(result.body))
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
						name := strings.ToLower(invalidChars.ReplaceAllLiteralString("itm_"+attGroup+"_"+items.Items[i].Properties[j].ID, "_"))
						desc := prometheus.NewDesc(name, "ITM metric "+items.Items[i].Properties[j].Label, nil, labelmap)
						value, err := strconv.ParseFloat(strings.Replace(items.Items[i].Properties[j].DisplayValue, ",", ".", -1), 64)
						log.Debugf("Group: %v | Name: %v | Labels: %v | Value: %v", attGroup, name, labelmap, value)
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
		log.Errorf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Errorf("Unmarshal: %v", err)
	}
	return c
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	var itmServerURL string

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	kingpin.HelpFlag.Short('h')
	if *itmServer != "" {
		itmServerURL = *itmServer
	} else {
		itmServerURL = conf.ItmServerURL
	}

	validate = validator.New()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case listAgentTypes.FullCommand():
		var datasource Datasource
		table.SetHeader([]string{"Agent Type", "Dataset URI"})

		responseBody, statusCode, err := MakeRequest(itmServerURL + "/ibm/tivoli/rest/providers/itm." + url.QueryEscape(*listAgentTypesTEMS) + "/datasources")
		if err == nil && statusCode == 200 {
			json.Unmarshal([]byte(responseBody), &datasource)
			for i := 0; i < len(datasource.Items); i++ {
				table.Append([]string{datasource.Items[i].Label, datasource.Items[i].DatasetsURI})
			}
			table.Render()
		} else {
			log.Error("Unable to collect Agent Types. Check your settings.")
		}
		os.Exit(0)
	case listAttributeGroups.FullCommand():
		var dataset Dataset
		var column Columns
		table.SetHeader([]string{"Description", "Attribute Group"})
		responseBody, statusCode, err := MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributeGroupsDataset)
		if err == nil && statusCode == 200 {
			json.Unmarshal([]byte(responseBody), &dataset)
			for i := 0; i < len(dataset.Items); i++ {
				if *listAGLong == true {
					if *listAttributesGroup == "msys" {
						responseBody, statusCode, err = MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributesDataset + "/msys/columns")
					} else {
						responseBody, statusCode, err = MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributeGroupsDataset + "/" + dataset.Items[i].ID + "/columns")
					}
					if err == nil && statusCode == 200 {
						json.Unmarshal([]byte(responseBody), &column)
						var s []string
						for j := 0; j < len(column.Items); j++ {
							s = append(s, strings.Replace(column.Items[j].ID, "MetricGroup.", "", -1))
						}
						attrList := strings.Join(s, "\", \"")
						fmt.Printf("%v|%v|\"%s\"\n", strings.Replace(dataset.Items[i].Label, "\n", "", -1), strings.Replace(dataset.Items[i].ID, "MetricGroup.", "", -1), attrList)
					}
				} else {
					table.Append([]string{strings.Replace(dataset.Items[i].Label, "\n", "", -1),
						strings.Replace(dataset.Items[i].ID, "MetricGroup.", "", -1)})
				}
			}
			if *listAGLong != true {
				table.Render()
			}
		} else {
			log.Error("Unable to collect Attribute Groups. Check your settings.")
		}
		os.Exit(0)
	case listAttributes.FullCommand():
		var column Columns
		var responseBody []byte
		var err error
		var statusCode int

		table.SetHeader([]string{"Description", "Attributes", "Primary Key"})
		if *listAttributesGroup == "msys" {
			responseBody, statusCode, err = MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributesDataset + "/msys/columns")
		} else {
			responseBody, statusCode, err = MakeRequest(itmServerURL + "/ibm/tivoli/rest" + *listAttributesDataset + "/MetricGroup." + *listAttributesGroup + "/columns")
		}
		if err == nil && statusCode == 200 {
			json.Unmarshal([]byte(responseBody), &column)
			for i := 0; i < len(column.Items); i++ {
				table.Append([]string{strings.Replace(column.Items[i].Label, "\n", "", -1),
					strings.Replace(column.Items[i].ID, "MetricGroup.", "", -1), strconv.FormatBool(column.Items[i].PrimaryKey)})
			}
			table.Render()
		} else {
			log.Error("Unable to collect Attributes. Check your settings.")
		}
		os.Exit(0)
	case export.FullCommand():
		if *verbose {
			log.SetLevel(log.DebugLevel)
		}
		if *itmServerPassword != "" {
			log.Error("For export mode you have to specify password in the config file.")
			os.Exit(0)
		}
		log.Info("Starting itm_exporter in export mode...")
		log.Info("Author: Rafal Szypulka")
	case test.FullCommand():
		if *verbose {
			log.SetLevel(log.DebugLevel)
		}
		diag = true
		log.Info("Starting itm_exporter in diagnostic mode...")
	}
	log.Debug("Verbose mode")
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
