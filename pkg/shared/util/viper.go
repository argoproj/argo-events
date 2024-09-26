package util

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

func ViperWithLogging() *viper.Viper {
	v := viper.NewWithOptions(viper.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))))
	return v
}
