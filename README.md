# Prometheus SUSE Manager Service Discovery

This tool connects to SUSE Manager servers and generates Prometheus scrape target configurations, taking advantage of the [file-based service discovery](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#%3Cfile_sd_config) mechanism provided by Prometheus.

## Prometheus Configuration

Please remember to adjust your `prometheus.yml` configuration file to use the file service discovery mechanism and point it to the output location of this tool.

Example configuration section of prometheus.yml:
```yaml
- job_name: 'overwritten-default'
  file_sd_configs:
   - files: ['/data/prometheus/scrape-config/*.yml']
```

## License

This project has been released under the MIT license. Please see the LICENSE.md file for more details.
