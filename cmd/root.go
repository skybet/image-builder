// Copyright (c) 2017 Adam Pointer

package cmd

import (
	"fmt"
	"os"

	"github.com/adampointer/image-builder/lib"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Setup logging
		if viper.GetBool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		if viper.GetBool("json") {
			log.SetFormatter(&log.JSONFormatter{})
		}
	},
	Use:   "image-builder",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet("git-url") {
			log.Fatal("--git-url must be set")
		}

		git := lib.Git()
		git.SetUrl(viper.GetString("git-url"))

		// First set up an in-memory clone
		if viper.IsSet("key-path") {
			if err := git.SetKey(viper.GetString("key-path")); err != nil {
				log.Fatalf("Error reading private key: %s", err)
			}
		}

		if err := git.Clone(viper.GetString("git-branch")); err != nil {
			log.Fatalf("Error cloning repository: %s", err)
		}

		// We get the latest commit and diff it with it's parent commit(s) to get a list of
		// paths which have changed. After popping off the filename and deduping, we have a list of
		// directories containing changes
		log.Info("Searching latest commit for changes to Docker repositories")
		dirs, err := git.DirsChanged()
		if err != nil {
			log.Fatalf("Error looking for changes: %s", err)
		}
		log.Debugf("Found changed dirs: %v", dirs)

		// Remove any directory without a Dockerfile to leave a list of Docker repositories which
		// need new images building
		roots, err := git.RemoveNonBuildPaths(dirs)
		if err != nil {
			log.Fatalf("Error filtering out directories without a Dockerfile: %s", err)
		}

		docker, err := lib.GetDockerClient(viper.GetString("docker-host"))
		if err != nil {
			log.Fatalf("Error creating docker client: %s", err)
		}

		if len(roots) > 0 {

			log.Info("Creating tar archives to send to Docker daemon")
			for _, p := range roots {
				log.Info(p)
				tar, err := git.GetTarAtPath(p)
				if err != nil {
					log.Fatalf("Error getting tar archive at %s: %s", p, err)
				}

				tags := git.GenerateTags(p)
				log.Infof("Building %s", p)
				if err := docker.Build(tar, tags); err != nil {
					log.Fatalf("Error building image at %s: %s", p, err)
				}

				log.Infof("Pushing %v", tags)
				if err := docker.Push(p, viper.GetString("docker-auth")); err != nil {
					log.Fatalf("Error pushing image at %s: %s", p, err)
				}
			}
		} else {
			log.Info("No changes found")
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	var (
		debug, json                            bool
		gitUrl, gitBranch, keyPath, dockerHost, dockerAuth string
	)

	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.image-builder.yaml)")

	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Debug mode")
	viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))
	viper.SetDefault("debug", false)

	RootCmd.PersistentFlags().BoolVarP(&json, "json", "j", false, "Log in json format")
	viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json"))
	viper.SetDefault("json", false)

	RootCmd.PersistentFlags().StringVarP(&gitUrl, "git-url", "g", "", "Git repo to build")
	viper.BindPFlag("git-url", RootCmd.PersistentFlags().Lookup("git-url"))

	RootCmd.PersistentFlags().StringVarP(&gitBranch, "git-branch", "b", "master", "Git branch to build")
	viper.BindPFlag("git-branch", RootCmd.PersistentFlags().Lookup("git-branch"))
	viper.SetDefault("git-branch", "master")

	RootCmd.PersistentFlags().StringVarP(&keyPath, "key-path", "k", "", "Path to private key")
	viper.BindPFlag("key-path", RootCmd.PersistentFlags().Lookup("key-path"))

	RootCmd.PersistentFlags().StringVarP(&dockerHost, "docker-host", "u", "unix:///var/run/docker.sock", "Docker host/socket")
	viper.BindPFlag("docker-host", RootCmd.PersistentFlags().Lookup("docker-host"))
	viper.SetDefault("docker-host", "unix:///var/run/docker.sock")

	RootCmd.PersistentFlags().StringVarP(&dockerAuth, "docker-auth", "a", "abc", "base64 encoded credentials for the registry")
	viper.BindPFlag("docker-auth", RootCmd.PersistentFlags().Lookup("docker-auth"))
	viper.SetDefault("docker-auth", "abc")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".image-builder")
		viper.AddConfigPath("$HOME")
	}
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ib")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
