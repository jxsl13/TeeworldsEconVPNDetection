package cmd

import (
	"context"
	"fmt"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
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
		Use:          "add",
		Short:        "add ips to the database (blacklist)",
		SilenceUsage: true,
		RunE:         addContext.RunE,
		Args:         cobra.ExactArgs(1),
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
	Ctx      context.Context
	Config   *config.ConnectConfig
	Ripr     *goripr.Client
	FilePath string
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

		c.FilePath = args[0]
		return nil
	}
}

func (c *addContext) RunE(cmd *cobra.Command, args []string) error {
	added, err := parseFileAndAddIPsToCache(
		c.Ctx,
		c.Ripr,
		c.FilePath,
	)
	if err != nil {
		return err
	}
	fmt.Printf("added %d ip ranges to the database\n", added)
	return nil
}
