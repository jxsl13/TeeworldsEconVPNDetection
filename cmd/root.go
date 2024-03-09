package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/jxsl13/TeeworldsEconVPNDetection/config"
	"github.com/jxsl13/TeeworldsEconVPNDetection/econ"
	"github.com/jxsl13/TeeworldsEconVPNDetection/vpn"
	"github.com/jxsl13/goripr/v2"
	"github.com/nutsdb/nutsdb"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	rootContext := rootContext{
		Ctx:    ctx,
		Config: config.New(),
	}

	// cmd represents the run command
	cmd := &cobra.Command{
		Use:          "TeeworldsEconVPNDetection",
		Short:        "TeeworldsEconVPNDetection is a tool for detecting and banning VPN user on a Teeworlds server.",
		SilenceUsage: true,
		RunE:         rootContext.RunE,
		Args:         cobra.ExactArgs(0),
		PostRunE: func(cmd *cobra.Command, args []string) error {
			cancel()
			return nil
		},
	}

	// register flags but defer parsing and validation of the final values
	cmd.PreRunE = rootContext.PreRunE(cmd)

	// register flags but defer parsing and validation of the final values
	cmd.AddCommand(NewCompletionCmd(cmd.Name()))
	cmd.AddCommand(NewAddCmd(ctx))
	cmd.AddCommand(NewRemoveCmd(ctx))
	return cmd
}

type rootContext struct {
	Ctx     context.Context
	Config  *config.Config
	Ripr    *goripr.Client
	Checker *vpn.VPNChecker
}

func (c *rootContext) PreRunE(cmd *cobra.Command) func(cmd *cobra.Command, args []string) error {

	runParser := config.RegisterFlags(
		c.Config,
		false,
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

		var wl *vpn.Whitelister
		bucket := c.Config.NutsDBBucket
		if !c.Config.Offline {
			// only needed for whitelisting non-vpn users
			nuts, err := nutsdb.Open(
				nutsdb.DefaultOptions,
				nutsdb.WithRWMode(nutsdb.MMap),
				nutsdb.WithDir(c.Config.NutsDBDir),
				nutsdb.WithSegmentSize(1024*1024), // 1MB
			)
			if err != nil {
				return err
			}

			err = nuts.Update(func(tx *nutsdb.Tx) error {
				if !tx.ExistBucket(nutsdb.DataStructureBTree, bucket) {
					return tx.NewKVBucket(bucket)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to create bucket: %w", err)
			}

			wl = vpn.NewWhitelister(nuts, bucket, c.Config.WhitelistTTL)
		}

		checker := vpn.NewVPNChecker(
			c.Ctx,
			ripr,
			wl,
			c.Config.APIs(),
			c.Config.Offline,
			c.Config.BanThreshold,
		)
		c.Checker = checker

		return nil
	}
}

func (c *rootContext) RunE(cmd *cobra.Command, args []string) error {
	log.Println("Starting up...")

	for _, file := range c.Config.Blacklists {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		log.Println("Adding blacklist file: ", file)
		added, err := parseFileAndAddIPsToCache(c.Ctx, c.Ripr, file)
		if err != nil {
			return err
		}
		log.Printf("Added %d ip ranges from %s\n", added, file)
	}

	for _, file := range c.Config.Whitelists {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		log.Println("Removing whitelist file: ", file)
		removed, err := parseFileAndRemoveIPsFromCache(c.Ctx, c.Ripr, file)
		if err != nil {
			return err
		}
		log.Printf("Removed %d ip ranges from %s\n", removed, file)
	}

	for idx, addr := range c.Config.EconServers {
		go econ.NewEvaluationRoutine(
			c.Ctx,
			addr,
			c.Config.EconPasswords[idx],
			c.Checker,
			c.Config.ReconnectDelay,
			c.Config.ReconnectTimeout,
			c.Config.VPNBanTime,
			c.Config.VPNBanReason,
		)
	}
	log.Println("Started up successfully")
	<-c.Ctx.Done()
	log.Println("Shutting down...")

	return nil
}
