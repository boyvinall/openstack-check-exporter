# openstack-check-exporter

![](docs/wip.jpg)

This exporter runs some active checks against an OpenStack cloud and exposes the results as Prometheus metrics. The checks are inspired by
the (now deprecated) nagios/sensu checks at
[osops-tools-monitoring/monitoring-for-openstack/oschecks](https://github.com/openstack-archive/osops-tools-monitoring/tree/7427ee739296e93f18aed92f7150abf732fd92b3/monitoring-for-openstack/oschecks)
and
[osops-tools-monitoring/nagios-plugins](https://github.com/openstack-archive/osops-tools-monitoring/tree/7427ee739296e93f18aed92f7150abf732fd92b3/nagios-plugins).

By comparison, <https://github.com/openstack-exporter/openstack-exporter> does not attempt to create any resources, merely
reading from the API and exposing those details.  It's recommended to run both of these exporters at the same time.
There is some overlapping functionality, though a benefit of the exporter from this repo is that durations are recorded,
which can be useful to monitor for performance regressions.
