# Kamaji Cluster API Control Plane provider

The Kamaji Control Plane provider implementation of the Cluster Management API.

## What is Kamaji?

[Kamaji](http://github.com/clastix/kamaji) is an Open-Source project offering hosted Kubernetes control planes.
tl;dr; the Control Plane is running in a management cluster as regular pods.

You can refer to the [official documentation website](https://kamaji.clastix.io/).

## Supported CAPI infrastructure providers

| Infrastructure Provider                                                        | Version       |
|--------------------------------------------------------------------------------|---------------|
| [OpenStack](https://github.com/kubernetes-sigs/cluster-api-provider-openstack) | += 0.7.4      |
| Metal³                                                                         | _In road-map_ |
| Equinix/Tinkerbell                                                             | _In road-map_ |
| vSphere                                                                        | _In road-map_ |
| Nutanix                                                                        | _In road-map_ |

> Are you looking for further integrations?
> Please, engage with the community on the [#kamaji](https://kubernetes.slack.com/archives/C03GLTTMWNN) Kubernetes Slack
> workspace channel, or using the **GitHub Discussion** section.

## Supported Kamaji version

The Cluster API Kamaji Control Plane provider is expecting to work with any Kamaji installation providing
the `tenantcontrolplanes.kamaji.clastix.io/v1alpha1` version, starting from
the [v0.3.0](https://github.com/clastix/kamaji/releases/tag/v0.3.0) release.
