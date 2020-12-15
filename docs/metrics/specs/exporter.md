## Proposal to use node-exporter
This document proposes the using of the industrial standard [node-exporter](https://github.com/prometheus/node_exporter). This light-weight utility can run on zos node and it will (out of the box) provide 100's of metrics.

This can also be extended easily to add more metrics that is ZOS specific like:
- reservations related metrics (workloads)

### Architecture
#### Node
- zos runs node-exporter
 - node exposes the exporter port only on the management (private) IP so only the farmer can access it
- we build a small utility (can be part of 3bot farm management) that generates prometheus config from the farm nodes
 - config contains basically node-ids and private IP
#### Farmer
- Run prometheus with provided configuration on an ubuntu node
- Run grafana
  - Download one of many node-exporter [pre-configured grafana dashboards](https://grafana.com/grafana/dashboards/1860)

## NOTICE
> In case we don't need a listener on ZOS (because that's one of the main zos reasons it's built this way to decrease the attack surface) we can change how node-exporter works to do a PUSH of metrics to local prometheus node instead of polling. Of course then the farmer has to boot the node with the right cmdline arguments. so the node knows where to push the data.

### UML
![uml](png/exporter.png)

### Dashboard example
![grafana-1](png/grafana-1.png)
![grafana-2](png/grafana-2.png)
