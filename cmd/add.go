package cmd

import (
	"context"
	"fmt"

	"github.com/jxsl13/TeeworldsEconVPNDetection/config"
	"github.com/jxsl13/goripr/v2"
	"github.com/spf13/cobra"
)

func NewAddCmd(ctx context.Context) *cobra.Command {

	addContext := addContext{
		Ctx:    ctx,
		Config: config.NewConnect(),
	}

	// cmd represents the run command
	cmd := &cobra.Command{
		Use:          "add blacklist.txt [more-banlists.txt...]",
		Short:        "add ips to the database (blacklist)",
		SilenceUsage: true,
		RunE:         addContext.RunE,
		Args:         cobra.MinimumNArgs(1),
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if addContext.Ripr != nil {
				return addContext.Ripr.Close()
			}
			return nil
		},
	}

	// register flags but defer parsing and validation of the final values
	cmd.PreRunE = addContext.PreRunE(cmd)
	return cmd
}

type addContext struct {
	Ctx       context.Context
	Config    *config.ConnectConfig
	Ripr      *goripr.Client
	FilePaths []string
}

func (c *addContext) PreRunE(cmd *cobra.Command) func(cmd *cobra.Command, args []string) error {
	runParser := config.RegisterFlags(
		c.Config,
		true,
		cmd,
		config.WithEnvPrefix("TWVPN_"),
	)
	return func(cmd *cobra.Command, args []string) error {
		err := runParser()
		if err != nil {
			return err
		}

		ripr, err := goripr.NewClient(
			c.Ctx,
			goripr.Options{
				Addr:     c.Config.RedisAddress,
				Password: c.Config.RedisPassword,
				DB:       c.Config.RedisDB,
			})
		if err != nil {
			return err
		}

		c.Ripr = ripr

		c.FilePaths = args
		return nil
	}
}

func (c *addContext) RunE(cmd *cobra.Command, args []string) error {
	for _, file := range c.FilePaths {
		fmt.Printf("adding ips from %s\n", file)
		added, err := parseFileAndAddIPsToCache(
			c.Ctx,
			c.Ripr,
			file,
		)
		if err != nil {
			return err
		}
		fmt.Printf("added %d ip ranges from %s to the database\n", added, file)
	}
	return nil
}
