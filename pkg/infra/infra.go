package infra

import (
	"strconv"

	compute "github.com/pulumi/pulumi-google-native/sdk/go/google/compute/v1"
	storage "github.com/pulumi/pulumi-google-native/sdk/go/google/storage/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/zxfsee/iac/pkg/config"
)

type Infrastructure struct {
	Bucket      *storage.Bucket
	Object      *storage.BucketObject
	Image       *compute.Image
	Group       *compute.InstanceGroup
	HealthCheck *compute.HealthCheck
	Backend     *compute.BackendService
	Proxy       *compute.TargetTcpProxy
	Ip          *compute.GlobalAddress
}

func CreateInfrastructure(ctx *pulumi.Context) (*Infrastructure, error) {
	cfg := config.GetConfig(ctx)
	bucket, err := storage.NewBucket(ctx, "bucket", &storage.BucketArgs{
		Location: cfg.Region,
		Name:     pulumi.String(config.BucketName),
		Project:  cfg.Project,
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("bucketSelfLink", bucket.SelfLink)

	object, err := storage.NewBucketObject(ctx, "object", &storage.BucketObjectArgs{
		Bucket: bucket.Name,
		Name:   pulumi.Sprintf("%s/%s", config.ImageName, cfg.ImageFile),
		Source: cfg.RemoteAsset,
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("objectSelfLink", object.SelfLink)

	image, err := compute.NewImage(ctx, "image", &compute.ImageArgs{
		Architecture: compute.ImageArchitectureArm64,
		GuestOsFeatures: compute.GuestOsFeatureArray{
			&compute.GuestOsFeatureArgs{
				Type: compute.GuestOsFeatureTypeVirtioScsiMultiqueue,
			},
		},
		Name:    pulumi.String(config.ImageName),
		Project: cfg.Project,
		RawDisk: &compute.ImageRawDiskArgs{
			Source: pulumi.Sprintf("https://storage.googleapis.com/%s/%s/%s", bucket.Name, config.ImageName, cfg.ImageFile),
		},
		StorageLocations: pulumi.StringArray{cfg.Region},
	}, pulumi.DependsOn([]pulumi.Resource{object}))
	if err != nil {
		return nil, err
	}
	ctx.Export("imageSelfLink", image.SelfLink)

	group, err := compute.NewInstanceGroup(ctx, "ig", &compute.InstanceGroupArgs{
		Name:    pulumi.Sprintf("%s-ig", config.ImageName),
		Project: cfg.Project,
		NamedPorts: compute.NamedPortArray{
			&compute.NamedPortArgs{
				Name: pulumi.String("tcp6443"),
				Port: pulumi.Int(config.Port),
			},
		},
		Zone: cfg.Zone,
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("groupSelfLink", group.SelfLink)

	healthcheck, err := compute.NewHealthCheck(ctx, "health-check", &compute.HealthCheckArgs{
		Name:    pulumi.Sprintf("%s-health-check", config.ImageName),
		Project: cfg.Project,
		Type:    compute.HealthCheckTypeTcp,
		TcpHealthCheck: &compute.TCPHealthCheckArgs{
			Port: pulumi.Int(config.Port),
		},
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("healthcheckSelfLink", healthcheck.SelfLink)

	backend, err := compute.NewBackendService(ctx, "be", &compute.BackendServiceArgs{
		Backends: compute.BackendArray{
			&compute.BackendArgs{
				Group: group.SelfLink,
			}},
		HealthChecks: pulumi.StringArray{healthcheck.SelfLink},
		Name:         pulumi.Sprintf("%s-be", config.ImageName),
		PortName:     pulumi.String("tcp6443"),
		Project:      cfg.Project,
		Protocol:     compute.BackendServiceProtocolTcp,
		TimeoutSec:   pulumi.Int(config.Timeout),
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("backendSelfLink", backend.SelfLink)

	proxy, err := compute.NewTargetTcpProxy(ctx, "tcp-proxy", &compute.TargetTcpProxyArgs{
		Name:        pulumi.Sprintf("%s-tcp-proxy", config.ImageName),
		Project:     cfg.Project,
		ProxyHeader: compute.TargetTcpProxyProxyHeaderNone,
		Service:     backend.SelfLink,
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("proxySelfLink", proxy.SelfLink)

	ip, err := compute.NewGlobalAddress(ctx, "lb-ip", &compute.GlobalAddressArgs{
		Name:    pulumi.Sprintf("%s-lb-ip", config.ImageName),
		Project: cfg.Project,
	})
	if err != nil {
		return nil, err
	}
	ctx.Export("ipAddress", ip.Address)

	_, err = compute.NewGlobalForwardingRule(ctx, "fwd-rule", &compute.GlobalForwardingRuleArgs{
		IpAddress: ip.Address,
		Name:      pulumi.Sprintf("%s-fwd-rule", config.ImageName),
		PortRange: pulumi.String("443"),
		Project:   cfg.Project,
		Target:    proxy.SelfLink,
	})
	if err != nil {
		return nil, err
	}

	_, err = compute.NewFirewall(ctx, "controlplane-firewall", &compute.FirewallArgs{
		Allowed: compute.FirewallAllowedItemArray{
			&compute.FirewallAllowedItemArgs{
				IpProtocol: pulumi.String("tcp"),
				Ports: pulumi.StringArray{
					pulumi.String(strconv.Itoa(config.Port)),
				},
			},
		},
		Name: pulumi.Sprintf("%s-controlplane-firewall", config.ImageName),
		SourceRanges: pulumi.StringArray{
			pulumi.String("130.211.0.0/22"),
			pulumi.String("35.191.0.0/16"),
		},
		TargetTags: pulumi.StringArray{
			pulumi.Sprintf("%s-controlplane", config.ImageName),
		},
	})
	if err != nil {
		return nil, err
	}

	return &Infrastructure{
		Bucket:      bucket,
		Object:      object,
		Image:       image,
		Group:       group,
		HealthCheck: healthcheck,
		Backend:     backend,
		Proxy:       proxy,
		Ip:          ip,
	}, nil
}
