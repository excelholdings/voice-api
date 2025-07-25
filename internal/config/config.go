package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	OpenAIAPIKey   string
	Port           string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPass         string
	DBName         string
	Env            string
	JWTSecret      string
	DeepgramAPIKey string
	ElevenLabsAPIKey string
	TwilioAccountSid string
	TwilioAccountAuthToken string
	TwilioStreamingURL string
	TwilioMLSid string
	TwilioMLUrl string
	FireworksAPIKey string
	ForwardRedirectMLUrl string

	StripeSecretKey string

	CartesiaAPIKey string
	CartesiaVersion string
}

func NewConfig() (*Config, error) {
	// Look for a .env file
	viper.SetConfigFile(".env")

	// Read the configuration from the .env file
	err := viper.ReadInConfig()

	// Use AutomaticEnv to override configuration with environment variables
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("OPENAI_API_KEY", "<placeholder>")
	viper.SetDefault("PORT", "<placeholder>")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASS", "password")
	viper.SetDefault("DB_NAME", "flyflow")
	viper.SetDefault("ENV", "local")
	viper.SetDefault("JWT_SECRET", "<placeholder>")
	viper.SetDefault("ELEVENLABS_API_KEY", "<placeholder>")
	viper.SetDefault("TWILIO_SID", "<placeholder>")
	viper.SetDefault("TWILIO_AUTH_TOKEN", "<placeholder>")
	viper.SetDefault("TWILIO_STREAMING_URL", "<placeholder>")
	viper.SetDefault("DEEPGRAM_API_KEY", "<placeholder>")
	viper.SetDefault("TWILIO_ML_SID", "<placeholder>")
	viper.SetDefault("TWILIO_ML_URL", "<placeholder>")
	viper.SetDefault("FIREWORKS_API_KEY", "<placeholder>")
	viper.SetDefault("TWILIO_REDIRECT_ML_URL", "<placeholder>")
	viper.SetDefault("STIPE_SECRET_KEY", "<placeholder>")
	viper.SetDefault("CARTESIA_API_KEY", "<placeholder>")
	viper.SetDefault("CARTESIA_VERSION", "<placeholder>")

	// Return the config
	return &Config{
		OpenAIAPIKey: viper.GetString("OPENAI_API_KEY"),
		Port:         viper.GetString("PORT"),
		DBHost:       viper.GetString("DB_HOST"),
		DBPort:       viper.GetString("DB_PORT"),
		DBUser:       viper.GetString("DB_USER"),
		DBPass:       viper.GetString("DB_PASS"),
		DBName:       viper.GetString("DB_NAME"),
		Env:          viper.GetString("ENV"),
		JWTSecret:    viper.GetString("JWT_SECRET"),
		DeepgramAPIKey: viper.GetString("DEEPGRAM_API_KEY"),
		ElevenLabsAPIKey: viper.GetString("ELEVENLABS_API_KEY"),
		TwilioAccountSid: viper.GetString("TWILIO_SID"),
		TwilioAccountAuthToken: viper.GetString("TWILIO_AUTH_TOKEN"),
		TwilioStreamingURL: viper.GetString("TWILIO_STREAMING_URL"),
		TwilioMLSid: viper.GetString("TWILIO_ML_SID"),
		TwilioMLUrl: viper.GetString("TWILIO_ML_URL"),
		FireworksAPIKey: viper.GetString("FIREWORKS_API_KEY"),
		ForwardRedirectMLUrl: viper.GetString("TWILIO_REDIRECT_ML_URL"),
		StripeSecretKey: viper.GetString("STIPE_SECRET_KEY"),
		CartesiaAPIKey: viper.GetString("CARTESIA_API_KEY"),
		CartesiaVersion: viper.GetString("CARTESIA_VERSION"),
	}, err
}
