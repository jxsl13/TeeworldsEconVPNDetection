package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/econ"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/vpn"
	"github.com/jxsl13/goripr/v2"
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
		Use:          "TeeworldsEconVPNDetectionGo",
		Short:        "TeeworldsEconVPNDetectionGo is a tool for detecting and banning VPN user on a Teeworlds server.",
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

		checker := vpn.NewVPNChecker(
			c.Ctx,
			ripr,
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

	for _, file := range c.Config.Whitelists {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		log.Println("Adding whitelist file: ", file)
		added, err := parseFileAndAddIPsToCache(c.Ctx, c.Ripr, file)
		if err != nil {
			return err
		}
		log.Printf("Added %d ip ranges from %s\n", added, file)
	}

	for _, file := range c.Config.Blacklists {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		log.Println("Removing blacklist file: ", file)
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
