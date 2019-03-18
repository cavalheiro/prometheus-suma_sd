package main

import (
  "flag"
  "io/ioutil"
  "fmt"
  "os"
  "time"
  "regexp"
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
  Groups           []GroupConfig
}

type GroupConfig struct {
  Labels          map[string]string
  Hosts           []HostConfig
}

type HostConfig struct {
  Match           string
  Ports           []string
  Labels          map[string]string
}

// Result structure
type PromScrapeGroup struct {
  Targets         []string
  Labels          map[string]string
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

// Get a list of SUSEManager client hostnames
func listSUSEManagerClients(config Config) ([]string, error) {
  mgrHost := config.Host
  result := []string{}
  token, err := Login(mgrHost, config.User, config.Pass)
  if err != nil {
    fmt.Printf("ERROR - Unable to login to SUSE Manager API: %v\n", err)
    return nil, err;
  }
  clientList, err := GetClientList(mgrHost, token)
  if err != nil {
    fmt.Printf("ERROR - Unable to get list of clients: %v\n", err)
    return nil, err;
  }
  for _, client := range clientList {
    clientDetails, err := GetClientDetails(mgrHost, token, client.Id)
    if err != nil {
      fmt.Printf("ERROR - Unable to get client details: %v\n", err)
      continue;
    }
    result = append(result, clientDetails.Hostname)
  }
  Logout(mgrHost, token)
  return result, nil
}

// Generate scrape configuration for a given list of hosts
func generatePromConfig(config Config, hosts []string) (error) {
  promConfig := []PromScrapeGroup{}
  for _, groupConfig := range config.Groups {
    for _, hostConfig := range groupConfig.Hosts {
      // Declare base structure for prom scrape group
      promScrapeGroup := PromScrapeGroup{Targets: []string{}, Labels: map[string]string{}}
      // Build a list of hosts that match with the config regex
      matchedDomains := func (hosts []string, filterExpr string) ([]string) {
        result := []string{}
        for _, host := range hosts {
          match, _ := regexp.MatchString(filterExpr, host)
          if match {
            result = append(result, host)
          }
        }
        return result
      }(hosts, hostConfig.Match)
      if len(matchedDomains) > 0 {
        // Add group labels
        for k, v := range groupConfig.Labels {
            promScrapeGroup.Labels[k] = v
        }
        // Add domain labels
        for k, v := range hostConfig.Labels {
            promScrapeGroup.Labels[k] = v
        }
        // Add scrape targets and ports
        for _, host :=  range matchedDomains {
          for _, port := range hostConfig.Ports {
            promScrapeGroup.Targets = append(promScrapeGroup.Targets, host + ":" + port)
          }
        }
        promConfig = append(promConfig, promScrapeGroup)
      }
    }
  }
  ymlPromConfig, _ := yaml.Marshal(promConfig)
  return ioutil.WriteFile(config.OutputDir+"/suma-clients.yml", []byte(ymlPromConfig), 0644)
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

  // Loop infinitely in case there is a pooling internal, run once otherwise
  for {
    fmt.Printf("Querying SUSE Manager server...\n")
    hosts, err := listSUSEManagerClients(config)
    if err != nil {
      continue; // we want to try in the next iteration, should the error be temporary
    }
    err = generatePromConfig(config, hosts)
    if err != nil {
      fmt.Printf("ERROR - Unable to write Prometheus config file: %v\n", err)
    }
    if config.PollingInterval > 0 {
      time.Sleep(time.Duration(config.PollingInterval) * time.Second)
    } else {
      break
    }
  }
}
