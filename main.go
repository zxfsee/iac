package main

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := digitalocean.NewDroplet(ctx, "controller-0", &digitalocean.DropletArgs{
			Image:   pulumi.String("ubuntu-20-04-x64"),
			Region:  pulumi.String("fra1"),
			Size:    pulumi.String("s-1vcpu-1gb"),
			SshKeys: pulumi.StringArray{pulumi.String("31421109")},
		})
		if err != nil {
			return err
		}
		return nil
	})
}
