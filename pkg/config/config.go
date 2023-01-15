package config

import (
	"path/filepath"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const BucketName string = "nhwk"
const ImageName string = "talos"

const Port int = 6443
const Timeout int = 300

type Data struct {
	Release string
}

type Config struct {
	Project     pulumi.StringOutput `pulumi:"project"`
	Region      pulumi.StringOutput `pulumi:"region"`
	Zone        pulumi.StringOutput `pulumi:"zone"`
	Data        *Data
	ImageFile   string
	RemoteAsset pulumi.Asset `pulumi:"remoteAsset"`
}

func GetConfig(ctx *pulumi.Context) *Config {
	cfg := &Config{}
	conf := config.New(ctx, "google-native")
	cfg.Project = conf.RequireSecret("project")
	cfg.Region = conf.RequireSecret("region")
	cfg.Zone = conf.RequireSecret("zone")

	conf = config.New(ctx, "")
	conf.RequireSecretObject("data", &cfg.Data)
	cfg.ImageFile = filepath.Base(cfg.Data.Release)
	cfg.RemoteAsset = pulumi.NewRemoteAsset(cfg.Data.Release)
	return cfg
}
