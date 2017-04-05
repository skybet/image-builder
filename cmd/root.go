// Copyright (c) 2017 Adam Pointer

package cmd

import (
	"fmt"
	"os"

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
	var debug, json bool

	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.image-builder.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Debug mode")
	viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))
	viper.SetDefault("debug", false)
	RootCmd.PersistentFlags().BoolVarP(&json, "json", "j", false, "Log in json format")
	viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json"))
	viper.SetDefault("json", false)
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
