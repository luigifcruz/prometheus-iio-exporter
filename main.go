package main

import (
    "os"
    "fmt"
    "bufio"
    "strings"
    "strconv"
    "net/http"
    "io/ioutil"

    "github.com/alexflint/go-arg"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type SensorGauge struct {
    Name      string
    Filestub  string
    Gauge     prometheus.Gauge
}

type UserDefined struct {
    IsDebug       bool   `arg:"--debug" help:"turn on debug mode" default:"false"` 
    SensorsPrefix string `arg:"--iioPrefix" help:"prefix to the iio directory" default:"/sys/bus/iio/devices/iio:device0"`
    ServerPort    string `arg:"--port" help:"data port to be used" default:"2112"` 
}

var (
    version           string
    config            UserDefined
    gauges            []SensorGauge
    prometheusHandler http.Handler
)

func readValue(filepath string) float64 {
    readFile, err := os.Open(filepath)
    if err != nil {
        return 0.0
    }

    fileScanner := bufio.NewScanner(readFile)
    fileScanner.Split(bufio.ScanLines)

    var value float64 = 0
  
    for fileScanner.Scan() {
        value, _ = strconv.ParseFloat(fileScanner.Text(), 64)
    }
  
    readFile.Close()

    return value;
}

func parse(stub string) float64 {
    raw := readValue(fmt.Sprintf("%s/%s_raw", config.SensorsPrefix, stub))
    scale := readValue(fmt.Sprintf("%s/%s_scale", config.SensorsPrefix, stub))
    offset := readValue(fmt.Sprintf("%s/%s_offset", config.SensorsPrefix, stub))
    return scale * (raw + offset) / 1000.
} 

func registerGauges() {
    if config.IsDebug {
        fmt.Println("Registering all available sensor gauges.")
    }

    files, _ := ioutil.ReadDir(config.SensorsPrefix)
    // TODO: Gracefully handle this error. 

    for _, file := range files {
        if !strings.Contains(file.Name(), "raw") { 
            continue
        } 

        filestr := strings.Split(file.Name(), "_")
        name := strings.Join(filestr[1:len(filestr)-1], "_")
        filestub := strings.Join(filestr[:len(filestr)-1], "_")

        if config.IsDebug {
            fmt.Println("- Adding gauge for sensor", name)
        }

        gauges = append(gauges, SensorGauge{
            Name: name,
            Filestub: filestub,
            Gauge: promauto.NewGauge(prometheus.GaugeOpts{
                Name: fmt.Sprintf("rfsoc_%s", name),
                Help: fmt.Sprintf("RFSoC sensor reading for %s", name),
            }),
        })
    }
}

func updateGauges() {
    if config.IsDebug {
        fmt.Println("Updating gauges.")
    }

    for _, sensor := range gauges {
        sensor.Gauge.Set(parse(sensor.Filestub))
    }
}

func main() {
    arg.MustParse(&config)
            
    // Set version.
    version = "1.0.0"

    // Use homepage to serve metadata.
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
        fmt.Fprintf(w, "Welcome to the Prometheus Exporter for RFSoC!\n")
        fmt.Fprintf(w, "The values are available at /metrics.\n")
        fmt.Fprintf(w, "Version: %s\n", version)
    })

    // Create all sensor gauges for RFSoC.
    registerGauges()

    // Create default HTTP request handler for Prometheus.
	prometheusHandler = promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, 
        promhttp.HandlerFor(prometheus.DefaultGatherer, 
                            promhttp.HandlerOpts{}),
	)

    http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request){
        updateGauges()
        prometheusHandler.ServeHTTP(w, r)
    })

    // Initiate HTTP server.
    fmt.Printf("Starting Prometheus server at port %s...\n", config.ServerPort)
    http.ListenAndServe(fmt.Sprintf(":%s", config.ServerPort), nil)
}
