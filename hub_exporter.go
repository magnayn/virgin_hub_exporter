package main

import (
    "fmt"
 //   "io/ioutil"
    "time"
    "net/http"
 //   "strings"
 //   "encoding/json"
    "strconv"
    "os"
    "flag"
    "sync"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/prometheus/common/log"
    "github.com/prometheus/common/version"
    "github.com/PuerkitoBio/goquery"

)
type ChannelInfo struct {
    ID, Width, Mod, Interleave, Annex int
    Freq uint64
    Power float64
    SNR float64
    PreRSErrors, PostRSErrors int
    Timeouts int
}

var (
    showVersion = flag.Bool("Version", false, "Print version number and exit")
    listenAddress = flag.String("web.listen-address", ":9463", "IP And Port to expose metrics on")
    modemIP = flag.String("modemIP", "192.168.100.1", "IP address of modem")
    timeout = flag.Duration("timeout", 5*time.Second, "Timeout for scrapes")
    metricsPath = flag.String("web.telemetry-path", "/metrics", "Path to metrics")

    channels = make(map[int]ChannelInfo)
    uschannels = make(map[int]ChannelInfo)
)
const ( namespace = "hub2ac" )
func init() {
    flag.Usage = func() {
        fmt.Println("Usage: hub_exporter [ ... ] \nFlags:")
        flag.PrintDefaults()
    }
    prometheus.MustRegister(version.NewCollector("hub3_exporter"))
}

func start() {
    log.Infof("Starting Hub Exporter (Version: %s)", version.Info())



    exporter := ProExporter(*timeout)
    prometheus.MustRegister(exporter)
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<html><head><title>Virgin SuperHub Exporter: Ver ` + version.Info() + `</title></head>` +
                       `<body><h1>Virgin Media / UPC Hub 2ac Metrics exporter</h1>` +
                       `<p><a href="` + *metricsPath + `">Metrics</a></p>` +
                       `</body></html>`))
    })
    http.Handle(*metricsPath, promhttp.Handler())
    log.Infof("Metrics exposed at %s on %s", *metricsPath, *listenAddress)
    log.Fatal(http.ListenAndServe(*listenAddress, nil))

}

func (p *PrometheusExporter) Collect(ch chan<- prometheus.Metric) {
    p.mutex.Lock()
    defer p.mutex.Unlock()


    doc, err := goquery.NewDocument(fmt.Sprintf("http://%s/cgi-bin/VmRouterStatusDownstreamCfgCgi", *modemIP))
    if err != nil {
        //LOG
        fmt.Println(err)
    }
    doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
      
        // For each item found, get the band and title
        s.Find("td").Each(func(j int, t *goquery.Selection) { 
//            fmt.Printf("%d,%d: %s \n", i,j, t.Text())

            if j > 0 {
               index := j-1
            
                c := channels[index]
                if c.ID == 0 {
                   log.Debugln("Found new Channel", index)
                   channels[index] = ChannelInfo{ID: 99, Freq: 1, Width: 2, Mod: 3, Interleave: 4, Power: 5, Annex:6}
                } 

                value, _ := strconv.Atoi(t.Text())
                switch i {
                    case 0:
                        fv, _ := strconv.ParseUint(t.Text(),10,64)
                        c.Freq = fv
                    case 2:
                        c.ID = value
                    case 6:
                        fv, _ := strconv.ParseFloat(t.Text(),64)
                        c.Power = fv;
                    case 7:
                         fv, _ := strconv.ParseFloat(t.Text(),64)
                        c.SNR = fv
                    case 8:
                        c.PreRSErrors = value
                    case 9:
                        c.PostRSErrors = value
                    
                    default:
                }
                channels[index] = c

            }
        })
        //title := s.Find("i").Text()
   //     fmt.Printf("Review %d: %s \n", i, td)
    })


    // Upstream
    doc, err = goquery.NewDocument(fmt.Sprintf("http://%s/cgi-bin/VmRouterStatusUpstreamCfgCgi", *modemIP))
    if err != nil {
        //LOG
        fmt.Println(err)
    }
    doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
      
        // For each item found, get the band and title
        s.Find("td").Each(func(j int, t *goquery.Selection) { 
            fmt.Printf("%d,%d: %s \n", i,j, t.Text())

            if j > 0 {
               index := j-1
            
                c := uschannels[index]
                if c.ID == 0 {
                   log.Debugln("Found new Channel", index)
                   uschannels[index] = ChannelInfo{ID: 99, Freq: 1, Width: 2, Mod: 3, Interleave: 4, Power: 5, Annex:6}
                } 

                value, _ := strconv.Atoi(t.Text())
                switch i {
                    case 2:
                        fv, _ := strconv.ParseUint(t.Text(),10,64)
                        c.Freq = fv
                    case 1:
                        c.ID = value
                    case 7:
                        fv, _ := strconv.ParseFloat(t.Text(),64)
                        c.Power = fv;
                        c.Timeouts = 0
                    case 8:                        
                        c.Timeouts += value
                    case 9:                        
                        c.Timeouts += value
                    case 10:                        
                        c.Timeouts += value
                    case 11:                        
                        c.Timeouts += value

                    
                    default:
                }
                uschannels[index] = c

            }
        })
        //title := s.Find("i").Text()
   //     fmt.Printf("Review %d: %s \n", i, td)
    })

     for _,c := range channels{
          ch <- prometheus.MustNewConstMetric(p.downFrequency, prometheus.GaugeValue, float64(c.Freq), strconv.Itoa(c.ID))
          ch <- prometheus.MustNewConstMetric(p.downPower, prometheus.GaugeValue, c.Power, strconv.Itoa(c.ID))
          ch <- prometheus.MustNewConstMetric(p.downSNR, prometheus.GaugeValue, c.SNR, strconv.Itoa(c.ID))

          ch <- prometheus.MustNewConstMetric(p.downPreRSErrors, prometheus.GaugeValue, float64(c.PreRSErrors), strconv.Itoa(c.ID))
          ch <- prometheus.MustNewConstMetric(p.downPostRSErrors, prometheus.GaugeValue, float64(c.PostRSErrors), strconv.Itoa(c.ID))
     }
     for _,c := range uschannels{
          // Frequency needs another grab from http://192.168.100.1/walk?oids=1.3.6.1.2.1.10.127.1.1.2;&_n=86936&_=1507125744036
          ch <- prometheus.MustNewConstMetric(p.upPower, prometheus.GaugeValue, c.Power, strconv.Itoa(c.ID))
          ch <- prometheus.MustNewConstMetric(p.upTimeouts, prometheus.GaugeValue, float64(c.Timeouts), strconv.Itoa(c.ID))
     }
}

type PrometheusExporter struct {
    mutex sync.Mutex

    downFrequency   *prometheus.Desc
    downPower   *prometheus.Desc
    downSNR  *prometheus.Desc

    downPreRSErrors *prometheus.Desc
    downPostRSErrors *prometheus.Desc


    upFrequency  *prometheus.Desc
    upPower  *prometheus.Desc
    upTimeouts  *prometheus.Desc
}

func ProExporter(timeout time.Duration) *PrometheusExporter {
    return &PrometheusExporter{
        downFrequency: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "downstream", "frequency_hertz"),
            "Downstream Frequency in HZ",
            []string{"channel"},
            nil,
        ),
        downPower: prometheus.NewDesc(
           prometheus.BuildFQName(namespace, "downstream", "power_dbmv"),
           "Downstream Power level in dBmv",
           []string{"channel"},
           nil,
        ),
        downSNR: prometheus.NewDesc(
           prometheus.BuildFQName(namespace, "downstream", "snr_db"),
           "Downstream SNR in dB",
           []string{"channel"},
           nil,
        ),
        downPreRSErrors: prometheus.NewDesc(
           prometheus.BuildFQName(namespace, "downstream", "pre_rs_errors"),
           "Pre RS Errors",
           []string{"channel"},
           nil,
        ),
        downPostRSErrors: prometheus.NewDesc(
           prometheus.BuildFQName(namespace, "downstream", "post_rs_errors"),
           "Post RS Errors",
           []string{"channel"},
           nil,
        ),
        upFrequency: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "upstream", "frequency_hertz"),
            "Upstream Frequency in HZ",
            []string{"channel"},
            nil,
        ),
        upPower: prometheus.NewDesc(
           prometheus.BuildFQName(namespace, "upstream", "power_dbmv"),
           "Upstream Power level in dBmv",
           []string{"channel"},
           nil,
        ),
        upTimeouts: prometheus.NewDesc(
           prometheus.BuildFQName(namespace, "upstream", "timeouts"),
           "Upstream sum of timeouts (T1-4)",
           []string{"channel"},
           nil,
        ),
    }
}

func (p *PrometheusExporter) Describe(ch chan<- *prometheus.Desc) {
    ch <- p.downFrequency
    ch <- p.upFrequency
    ch <- p.downPower
    ch <- p.upPower
    ch <- p.downSNR
}


func main() {
    flag.Parse()
    if *showVersion {
        os.Exit(0)
    }
    start()
}