package main

import (
  "flag"
  "io/ioutil"
  "fmt"
  "os"
  "time"
  "gopkg.in/yaml.v2"
)

const DEFAULT_CONFIG_FILE = "prometheus-suma_sd.yml"

// ------------
//  Structures
// ------------

type Config struct {
  OutputDir        string
  PollingInterval  int
  Host             string
  User             string
  Pass             string
}

// Result structure
type PromScrapeGroup struct {
  Targets         []string
  // Labels          map[string]string
}

// ------------------
//  Helper functions
// ------------------

// Error handler
func fatalErrorHandler(e error, msg string) {
  if e != nil {
    fmt.Printf("ERROR: %s\n", e.Error())
    fmt.Printf("ERROR: %s\n", msg)
    os.Exit(1)
  }
}


// Generate Scrape targets for SUMA client systems
func writePromConfigForClientSystems(config Config) (error) {
  apiUrl := "http://" + config.Host + "/rpc/api"
  targets := []string{}
  token, err := Login(apiUrl, config.User, config.Pass)
  if err != nil {
    fmt.Printf("ERROR - Unable to login to SUSE Manager API: %v\n", err)
    return err;
  }
  clientList, err := ListSystems(apiUrl, token)
  if err != nil {
    fmt.Printf("ERROR - Unable to get list of systems: %v\n", err)
    return err;
  }
  if len(clientList) == 0 {
    fmt.Printf("\tFound 0 systems.\n")
  } else {
    for _, client := range clientList {
      fqdns := []string{}
      formulas := formulaData{}
      details, err := GetSystemDetails(apiUrl, token, client.Id)
      if err != nil {
        fmt.Printf("ERROR - Unable to get system details: %v\n", err)
        continue;
      }
      // Check if system is to be monitored
      for _, v := range details.Entitlements {
        if v == "monitoring_entitled" {
          fqdns, err = ListSystemFQDNs(apiUrl, token, client.Id)
          formulas, err = getSystemFormulaData(apiUrl, token, client.Id, "prometheus-exporters")
          if (formulas.NodeExporter.Enabled) {
            targets = append (targets, fqdns[len(fqdns)-1] + ":9100")
          }
          if (formulas.PostgresExporter.Enabled) {
            targets = append (targets, fqdns[len(fqdns)-1] + ":9187")
          }
        }
      }
      fmt.Printf("\tFound system: %s, %v, FQDN: %v Formulas: %+v\n", details.Hostname, details.Entitlements, fqdns, formulas)
    }
  }
  Logout(apiUrl, token)
  promConfig := []PromScrapeGroup{PromScrapeGroup{Targets: targets}}
  ymlPromConfig, _ := yaml.Marshal(promConfig)
  return ioutil.WriteFile(config.OutputDir+"/suma-systems.yml", []byte(ymlPromConfig), 0644)
}

// Generate Scrape targets for SUMA server
func writePromConfigForSUMAServer(config Config) (error) {
  targets := []string{
    config.Host+":9100", // node_exporeter
    config.Host+":9187", // postgres_exporter
    config.Host+":5556", // jmx_exporter tomcat
    config.Host+":5557", // jmx_exporter taskomatic
    config.Host+":9800", // suma exporter
  }
  promConfig := []PromScrapeGroup{PromScrapeGroup{Targets: targets}}
  ymlPromConfig, _ := yaml.Marshal(promConfig)
  return ioutil.WriteFile(config.OutputDir+"/suma-server.yml", []byte(ymlPromConfig), 0644)
}

// ------
//  Main
// ------

func main() {
  // Parse command line arguments
  configFile := flag.String("config", DEFAULT_CONFIG_FILE, "Path to config file")
  flag.Parse()
  config := Config{PollingInterval: 120, OutputDir: "/tmp"} // Set defaults

  // Load configuration file
  dat, err := ioutil.ReadFile(*configFile)
  fatalErrorHandler(err, "Unable to read configuration file - please specify the correct location using --config=file.yml")
  err = yaml.Unmarshal([]byte(dat), &config)
  fatalErrorHandler(err, "Unable to parse configuration file")

  // Output some info about supplied config
  fmt.Printf("Using config file: %v\n", *configFile)
  fmt.Printf("\tSUSE Manager API URL: %v\n", config.Host)
  fmt.Printf("\tpolling interval: %d seconds\n", config.PollingInterval)
  fmt.Printf("\toutput dir: %v\n", config.OutputDir)

  // Generate config for SUSE Manager server (self-monitoring)
  writePromConfigForSUMAServer(config)
  // Loop infinitely in case there is a pooling internal, run once otherwise
  for {
    fmt.Printf("Querying SUSE Manager server API...\n")
    startTime := time.Now()
    err := writePromConfigForClientSystems(config)
    duration := time.Since(startTime)
    if err != nil {
      fmt.Printf("ERROR - Unable to generate config for client systems: %v\n", err)
    } else  {
      fmt.Printf("\tQuery took: %s\n", duration)
      fmt.Printf("Prometheus scrape target configuration updated.\n")
    }
    if config.PollingInterval > 0 {
      time.Sleep(time.Duration(config.PollingInterval) * time.Second)
    } else {
      break
    }
  }
}
