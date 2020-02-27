package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	input "github.com/natsukagami/go-input"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup of aws-cli-oidc",
	Long:  `Interactive setup of aws-cli-oidc. Will prompt you for OIDC provider URL and other settings.`,
	Run:   setup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func setup(cmd *cobra.Command, args []string) {
	runSetup()
}

func runSetup() {
	providerName, _ := ui.Ask("OIDC provider name:", &input.Options{
		Required: true,
		Loop:     true,
	})
	var authURL string
	var tokenURL string
	oidcServer, _ := ui.Ask("OIDC provider metadata server name (https://<server>/.well-known/openid-configuration):", &input.Options{
		Required: true,
		Loop:     true,
		ValidateFunc: func(s string) error {
			u, err := url.Parse(s)
			if err != nil {
				return err
			}

			u.Path = path.Join(u.Path, ".well-known", "openid-configuration")
			u.Scheme = "https"
			res, err := http.Get(u.String())
			if err != nil {
				return err
			}

			type oidcMetadata struct {
				AuthURL  string `json:"authorization_endpoint"`
				TokenURL string `json:"token_endpoint"`
			}

			bytes, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			var meta oidcMetadata
			if err := json.Unmarshal(bytes, &meta); err != nil {
				return err
			}

			authURL = meta.AuthURL
			tokenURL = meta.TokenURL
			return nil
		},
	})
	clientID, _ := ui.Ask("Client ID which is registered in the OIDC provider:", &input.Options{
		Required: true,
		Loop:     true,
	})
	clientSecret, _ := ui.Ask("Client secret which is registered in the OIDC provider (Default: none):", &input.Options{
		Default:  "",
		Required: false,
	})
	maxSessionDurationSeconds, _ := ui.Ask("The max session duration, in seconds, of the role session [900-43200] (Default: 3600):", &input.Options{
		Default:  "3600",
		Required: true,
		Loop:     true,
		ValidateFunc: func(s string) error {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil || i < 900 || i > 43200 {
				return fmt.Errorf("input must be 900-43200")
			}
			return nil
		},
	})

	config := map[string]string{}

	config[OIDCServer] = oidcServer
	config[AuthURL] = authURL
	config[TokenURL] = tokenURL
	config[ClientID] = clientID
	config[ClientSecret] = clientSecret
	config[MaxSessionDurationSeconds] = maxSessionDurationSeconds

	viper.Set(providerName, config)

	_ = os.MkdirAll(ConfigPath(), 0700)
	configPath := ConfigPath() + "/config.yaml"
	viper.SetConfigFile(configPath)
	err := viper.WriteConfig()

	if err != nil {
		log.Fatalf("Failed to write %s\n", configPath)
	}

	log.Printf("Saved %s\n", configPath)
}
