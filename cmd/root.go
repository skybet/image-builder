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
		fmt.Printf("%v\n", roots)
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
		debug, json                bool
		gitUrl, gitBranch, keyPath string
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