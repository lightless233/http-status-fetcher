package main

type AppConfig struct {
	Target     string
	InputFile  string
	OutputFile string

	TaskCount uint

	Debug bool
}

var appConfig AppConfig

func GetAppConfig() *AppConfig {
	return &appConfig
}
