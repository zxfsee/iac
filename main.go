package main

import (
	"fmt"
	"path/filepath"
	"strconv"

	compute "github.com/pulumi/pulumi-google-native/sdk/go/google/compute/v1"
	storage "github.com/pulumi/pulumi-google-native/sdk/go/google/storage/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	const bucketName string = "nhwk"
	const imageName string = "talos"

	const port int = 6443
	const timeout int = 300

	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "google-native")
		project := conf.Require("project")
		region := conf.Require("region")
		zone := conf.Require("zone")

		imageUrl := config.Require(ctx, fmt.Sprintf("%s:%s-release", ctx.Project(), imageName))
		imageFile := filepath.Base(imageUrl)
		remoteAsset := pulumi.NewRemoteAsset(imageUrl)
		// Create a Google Cloud resource (Storage Bucket)
		bucket, err := storage.NewBucket(ctx, "bucket", &storage.BucketArgs{
			Location: pulumi.String(region),
			Name:     pulumi.String(bucketName),
			Project:  pulumi.String(project),
		})
		if err != nil {
			return err
		}

		object, err := storage.NewBucketObject(ctx, "object", &storage.BucketObjectArgs{
			Bucket: bucket.Name,
			Name:   pulumi.Sprintf("%s/%s", imageName, imageFile),
			Source: remoteAsset,
		})
		if err != nil {
			return err
		}

		image, err := compute.NewImage(ctx, "image", &compute.ImageArgs{
			Architecture: compute.ImageArchitectureArm64,
			GuestOsFeatures: compute.GuestOsFeatureArray{
				&compute.GuestOsFeatureArgs{
					Type: compute.GuestOsFeatureTypeVirtioScsiMultiqueue,
				},
			},
			Name:    pulumi.String(imageName),
			Project: pulumi.String(project),
			RawDisk: &compute.ImageRawDiskArgs{
				Source: pulumi.Sprintf("https://storage.googleapis.com/%s/%s/%s", bucket.Name, imageName, imageFile),
			},
			StorageLocations: pulumi.StringArray{pulumi.String(region)},
		}, pulumi.DependsOn([]pulumi.Resource{object}))
		if err != nil {
			return err
		}

		group, err := compute.NewInstanceGroup(ctx, "ig", &compute.InstanceGroupArgs{
			Name:    pulumi.Sprintf("%s-ig", imageName),
			Project: pulumi.String(project),
			NamedPorts: compute.NamedPortArray{
				&compute.NamedPortArgs{
					Name: pulumi.String("tcp6443"),
					Port: pulumi.Int(port),
				},
			},
			Zone: pulumi.String(zone),
		})
		if err != nil {
			return err
		}

		healthcheck, err := compute.NewHealthCheck(ctx, "health-check", &compute.HealthCheckArgs{
			Name:    pulumi.Sprintf("%s-health-check", imageName),
			Project: pulumi.String(project),
			Type:    compute.HealthCheckTypeTcp,
			TcpHealthCheck: &compute.TCPHealthCheckArgs{
				Port: pulumi.Int(port),
			},
		})
		if err != nil {
			return err
		}

		backend, err := compute.NewBackendService(ctx, "be", &compute.BackendServiceArgs{
			Backends: compute.BackendArray{
				&compute.BackendArgs{
					Group: group.SelfLink,
				}},
			HealthChecks: pulumi.StringArray{healthcheck.SelfLink},
			Name:         pulumi.Sprintf("%s-be", imageName),
			PortName:     pulumi.String("tcp6443"),
			Project:      pulumi.String(project),
			Protocol:     compute.BackendServiceProtocolTcp,
			TimeoutSec:   pulumi.Int(timeout),
		})
		if err != nil {
			return err
		}

		proxy, err := compute.NewTargetTcpProxy(ctx, "tcp-proxy", &compute.TargetTcpProxyArgs{
			Name:        pulumi.Sprintf("%s-tcp-proxy", imageName),
			Project:     pulumi.String(project),
			ProxyHeader: compute.TargetTcpProxyProxyHeaderNone,
			Service:     backend.SelfLink,
		})
		if err != nil {
			return err
		}

		ip, err := compute.NewGlobalAddress(ctx, "lb-ip", &compute.GlobalAddressArgs{
			Name:    pulumi.Sprintf("%s-lb-ip", imageName),
			Project: pulumi.String(project),
		})
		if err != nil {
			return err
		}

		_, err = compute.NewGlobalForwardingRule(ctx, "fwd-rule", &compute.GlobalForwardingRuleArgs{
			IpAddress: ip.Address,
			Name:      pulumi.Sprintf("%s-fwd-rule", imageName),
			PortRange: pulumi.String("443"),
			Project:   pulumi.String(project),
			Target:    proxy.SelfLink,
		})
		if err != nil {
			return err
		}

		_, err = compute.NewFirewall(ctx, "controlplane-firewall", &compute.FirewallArgs{
			Allowed: compute.FirewallAllowedItemArray{
				&compute.FirewallAllowedItemArgs{
					IpProtocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String(strconv.Itoa(port)),
					},
				},
			},
			Name: pulumi.Sprintf("%s-controlplane-firewall", imageName),
			SourceRanges: pulumi.StringArray{
				pulumi.String("130.211.0.0/22"),
				pulumi.String("35.191.0.0/16"),
			},
			TargetTags: pulumi.StringArray{
				pulumi.Sprintf("%s-controlplane", imageName),
			},
		})
		if err != nil {
			return err
		}

		// Export the bucket self-link
		ctx.Export("bucketSelfLink", bucket.SelfLink)
		ctx.Export("objectSelfLink", object.SelfLink)
		ctx.Export("imageSelfLink", image.SelfLink)
		ctx.Export("groupSelfLink", group.SelfLink)
		ctx.Export("healthcheckSelfLink", healthcheck.SelfLink)
		ctx.Export("backendSelfLink", backend.SelfLink)
		ctx.Export("proxySelfLink", proxy.SelfLink)
		ctx.Export("ipAddress", ip.Address)
		return nil
	})
}
