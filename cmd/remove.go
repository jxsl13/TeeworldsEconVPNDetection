package cmd

import (
	"context"
	"fmt"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
	"github.com/jxsl13/goripr/v2"
	"github.com/spf13/cobra"
)

func NewRemoveCmd(ctx context.Context) *cobra.Command {

	removeContext := removeContext{
		Ctx:    ctx,
		Config: config.NewConnect(),
	}

	// cmd represents the run command
	cmd := &cobra.Command{
		Use:          "remove whitelist.txt [more-whitelists.txt...]",
		Short:        "remove ips from the database (whitelist)",
		SilenceUsage: true,
		RunE:         removeContext.RunE,
		Args:         cobra.MinimumNArgs(1),
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if removeContext.Ripr != nil {
				return removeContext.Ripr.Close()
			}
			return nil
		},
	}

	// register flags but defer parsing and validation of the final values
	cmd.PreRunE = removeContext.PreRunE(cmd)
	return cmd
}

type removeContext struct {
	Ctx       context.Context
	Config    *config.ConnectConfig
	Ripr      *goripr.Client
	FilePaths []string
}

func (c *removeContext) PreRunE(cmd *cobra.Command) func(cmd *cobra.Command, args []string) error {
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

func (c *removeContext) RunE(cmd *cobra.Command, args []string) error {
	for _, file := range c.FilePaths {
		fmt.Printf("removing ips from %s\n", file)
		removed, err := parseFileAndRemoveIPsFromCache(
			c.Ctx,
			c.Ripr,
			file,
		)
		if err != nil {
			return err
		}
		fmt.Printf("removed %d ip ranges from the database\n", removed)
	}
	return nil
}
