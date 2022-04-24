/*
Copyright © 2022 Michael Bruskov <mixanemca@yandex.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Targets []TargetsConfig `mapstructure:"targets"`
	Search  SearchConfig    `mapstructure:"search_config"`
}

type TargetsConfig struct {
	Name   string       `mapstructure:"name"`
	Type   string       `mapstructure:"type"`
	Notion NotionConfig `mapstructure:"notion_config"`
}

type SearchConfig struct {
	LastEditedTimeStart string `mapstructure:"lastEditedTimeStart"`
	LastEditedDateStart string `mapstructure:"lastEditedDateStart"`
	LastEditedTimeEnd   string `mapstructure:"lastEditedTimeEnd"`
	LastEditedDateEnd   string `mapstructure:"lastEditedDateEnd"`
}

type NotionConfig struct {
	APIKey          string        `mapstructure:"apiKey"`
	DatabaseID      string        `mapstructure:"databaseID"`
	UserID          string        `maspstructure:"userID"`
	Username        string        `mapstructure:"username"`
	Timeout         time.Duration `mapstructure:"timeout"`
	HeadingDoneName string        `mapstructure:"headingDoneName"`
	HeadingToDoName string        `mapstructure:"headingToDoName"`
}

func Init() (*Config, error) {
	// Command line flags
	pflag.StringP("starttime", "s", "24h", "Start time when notes was last updated.")
	pflag.StringP("startdate", "d", "", "Start date when notes was last updated.")
	pflag.StringP("endtime", "e", "", "End time when notes was last updated.")
	pflag.StringP("enddate", "j", "", "End date when notes was last updated.")
	pflag.Parse()

	// Bind command line flags
	_ = viper.BindPFlag("search_config.lastEditedTimeStart", pflag.Lookup("starttime"))
	_ = viper.BindPFlag("search_config.lastEditedDateStart", pflag.Lookup("startdate"))
	_ = viper.BindPFlag("search_config.lastEditedTimeEnd", pflag.Lookup("endtime"))
	_ = viper.BindPFlag("search_config.lastEditediDateEnd", pflag.Lookup("enddate"))

	// Name of config file (without extension)
	viper.SetConfigName(".taskgram")
	// REQUIRED if the config file does not have the extension in the name
	// path to look for the config file in call multiple times to add many
	// search paths
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")
	viper.AddConfigPath("$HOME/")
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			return nil, fmt.Errorf("config file not found")
		}
		return nil, fmt.Errorf("failed read config file: %v\n", err)
	}

	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
