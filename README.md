# openstack-check-exporter

![](docs/wip.jpg)

This exporter runs some active checks against an OpenStack cloud and exposes the results as Prometheus metrics. The checks are inspired by
the (now deprecated) nagios/sensu checks at
[osops-tools-monitoring/monitoring-for-openstack/oschecks](https://github.com/openstack-archive/osops-tools-monitoring/tree/7427ee739296e93f18aed92f7150abf732fd92b3/monitoring-for-openstack/oschecks)
and
[osops-tools-monitoring/nagios-plugins](https://github.com/openstack-archive/osops-tools-monitoring/tree/7427ee739296e93f18aed92f7150abf732fd92b3/nagios-plugins).

By comparison, <https://github.com/openstack-exporter/openstack-exporter> does not attempt to create any resources, merely reading from the
API and exposing those details.  It's recommended to run both of these exporters at the same time. There is some overlapping functionality,
though a benefit of the exporter from this repo is that durations are recorded, which can be useful to monitor for performance regressions.

## Notes

* This exporter runs the checks as a background process and then serves up the cached metrics for scraping.  This is contrary to conventions
  with prometheus - but, in this case, has some benefits:

    * It allows multiple prometheus servers to scrape the exporter without resulting in multiple resources being created. Specifically, this
      is important for things like creation of nova instances, which could add unnecessary load to the cloud.  
    * Additionally, the nova instance check ensures that there is not already a VM of the given name running. If the exporter is scraped
      multiple times then this would need to somehow pass the VM name in as a custom scrape query arg - doable, but a bit messy.

## To do

* [ ] CI, unit tests, etc
