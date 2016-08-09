package nginx

// Config holds NGINX configuration parameters
type Config struct {
	ProxyConnectTimeout           string
	ProxyReadTimeout              string
	ClientMaxBodySize             string
	MainServerNamesHashBucketSize string
	MainServerNamesHashMaxSize    string
}

// NewDefaultConfig creates a Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		ProxyConnectTimeout:        "60s",
		ProxyReadTimeout:           "60s",
		ClientMaxBodySize:          "1m",
		MainServerNamesHashMaxSize: "512",
		MainServerNamesHashBucketSize: "32",
	}
}
