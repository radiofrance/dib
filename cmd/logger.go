package main

import (
	"github.com/radiofrance/dib/internal/logger"
	"github.com/spf13/viper"
)

func initLogLevel() {
	_ = viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	logLevel := viper.GetString("log_level")
	logger.SetLevel(&logLevel)
}
