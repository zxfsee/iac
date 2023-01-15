package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/zxfsee/iac/pkg/infra"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := infra.CreateInfrastructure(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}
