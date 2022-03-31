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

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Notion NotionConfig `mapstructure:"notion"`
	Search SearchConfig `mapstructure:"search"`
}

type SearchConfig struct {
	LastEditedTimeStart string `mapstructure:"lastEditedTimeStart"`
	LastEditedTimeEnd   string `mapstructure:"lastEditedTimeEnd"`
	HeadingDoneName     string `mapstructure:"headingDoneName"`
	HeadingToDoName     string `mapstructure:"headingToDoName"`
}

type NotionConfig struct {
	APIKey     string `mapstructure:"apiKey"`
	DatabaseID string `mapstructure:"databaseID"`
	UserID     string `maspstructure:"userID"`
	Username   string `mapstructure:"username"`
	Timeout    string `mapstructure:"timeout"`
}

func Init() (*Config, error) {
	// Command line flags
	pflag.StringP("apikey", "a", "", "You Notion's API key.")
	pflag.StringP("databaseid", "d", "", "The Database UUID where you store notes.")
	pflag.StringP("username", "u", "", "Your preferred name in Notion account.")
	pflag.StringP("userid", "i", "", "Your Notion's user ID")
	pflag.StringP("timeout", "t", "10s", "Timeout for Notion's requests.")
	pflag.StringP("starttime", "s", "24h", "Start time when notes was last updated.")
	pflag.StringP("endtime", "e", "", "End time when notes was last updated.")
	pflag.StringP("doneblockkname", "n", "Workflow notes", "Name of heading block where you write done notes.")
	pflag.StringP("todoblockkname", "b", "TODO", "Name of heading block where you write ToDo notes.")
	pflag.Parse()

	// Bind command line flags
	viper.BindPFlag("notion.apiKey", pflag.Lookup("apikey"))
	viper.BindPFlag("notion.databaseID", pflag.Lookup("databaseid"))
	viper.BindPFlag("notion.username", pflag.Lookup("username"))
	viper.BindPFlag("notion.userID", pflag.Lookup("userid"))
	viper.BindPFlag("notion.timeout", pflag.Lookup("timeout"))
	viper.BindPFlag("search.lastEditedTimeStart", pflag.Lookup("starttime"))
	viper.BindPFlag("search.lastEditedTimeEnd", pflag.Lookup("endtime"))
	viper.BindPFlag("search.headingDoneName", pflag.Lookup("doneblockkname"))
	viper.BindPFlag("search.headingToDoName", pflag.Lookup("todoblockkname"))

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
