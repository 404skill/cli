package config

import (
    "os"
    "time"
    "context"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "fmt"
    "404skill-cli/auth"
    "404skill-cli/supabase"
)

func init() {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        panic("Unable to determine user home directory")
    }

    err = os.MkdirAll(fmt.Sprintf("%s/.404skill", homeDir), os.ModePerm)
    if err != nil {
        panic("Unable to create .404skill directory")
    }

    ConfigFilePath = fmt.Sprintf("%s/.404skill/config.yml", homeDir)
}

var ConfigFilePath string

type Config struct {
    Username    string `yaml:"username"`
    Password    string `yaml:"password"`
    AccessToken string `yaml:"access_token"`
    LastUpdated time.Time `yaml:"last_updated"`
}

func ReadConfig() (Config, error) {
    var config Config
    data, err := ioutil.ReadFile(ConfigFilePath)
    if err != nil {
        return config, err
    }
    err = yaml.Unmarshal(data, &config)
    return config, err
}

func WriteConfig(config Config) error {
    data, err := yaml.Marshal(&config)
    if err != nil {
        return err
    }
    return ioutil.WriteFile(ConfigFilePath, data, 0600)
}

func IsTokenExpired(lastUpdated time.Time) bool {
    return time.Since(lastUpdated) >= 1*time.Second
} 

func GetToken() (string, error) {
    config, err := ReadConfig()
    if err != nil {
        return "", err
    }

    if IsTokenExpired(config.LastUpdated) {
        client, err := supabase.NewSupabaseClient()
        if err != nil {
            fmt.Println(err)
            return "", err
        }

        authProvider := auth.NewSupabaseAuth(client)

        accessToken, err := authProvider.SignIn(context.Background(), config.Username, config.Password); 
        if err != nil {
            fmt.Println("Invalid credentials")
            return "", err
        }

        cfg := Config{
            Username:    config.Username,
            Password:    config.Password,
            AccessToken: accessToken,
            LastUpdated: time.Now(),
        }
    
        err = WriteConfig(cfg)
        if err != nil {
            return "", err
        }
    }

    return config.AccessToken, nil
}
