package main

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {
        _, err := digitalocean.NewDroplet(ctx, "web", &digitalocean.DropletArgs{
            Image: pulumi.String("ubuntu-20-04-x64"),
            Region: pulumi.String("nyc2"),
            Size: pulumi.String("s-1vcpu-1gb"),
        })
        if err != nil {
          return err
        }
        return nil
  })
}
