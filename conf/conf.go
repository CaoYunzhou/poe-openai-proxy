package conf

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type ConfigStruct struct {
	Port          int
	Tokens        []string
	Bot           map[string]string
	SimulateRoles int
	RateLimit     int
	CoolDown      int
	Timeout       int
	AuthKey		  string
	Proxy         string 
}

type ModelDef struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResp struct {
	Object string     `json:"object"`
	Data   []ModelDef `json:"data"`
}

var Conf ConfigStruct

var Models ModelsResp

func loadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}

func loadEnvVarAsInt(key string, defaultValue int) int {
	valueStr := loadEnvVar(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func loadEnvVarAsSlice(key string) []string {
	valueStr := loadEnvVar(key, "")
	return strings.Split(valueStr, ",")
}

func Setup() {
	// Load environment variables from .env file (for development purposes)
	_ = godotenv.Load()

	Conf.Port = loadEnvVarAsInt("PORT", 8080)
	Conf.Tokens = loadEnvVarAsSlice("TOKENS")
	Conf.AuthKey = loadEnvVarAsSlice("AuthKey")[0]
	Conf.SimulateRoles = loadEnvVarAsInt("SIMULATE_ROLES", 0)
	Conf.RateLimit = loadEnvVarAsInt("RATE_LIMIT", 10)
	Conf.CoolDown = loadEnvVarAsInt("COOL_DOWN", 10)
	Conf.Timeout = loadEnvVarAsInt("TIMEOUT", 60)
	Conf.Proxy = loadEnvVar("PROXY", "")

	Conf.Bot = map[string]string{
		"assistant":                    "capybara",
		"gpt-3.5-turbo":				"chinchilla",
		"gpt-3.5-turbo-0301":			"chinchilla",
		"gpt-3.5-turbo-0613":           "chinchilla",
		"gpt-3.5-turbo-16k":			"agouti",
		"gpt-3.5-turbo-16k-0613":       "agouti",
		"gpt-4":                        "beaver",
		"gpt-4-0314":                   "beaver",
		"gpt-4-0613":                   "beaver",
		"gpt-4-32k":                    "vizcacha",
		"gpt-4-32k-0314":				"vizcacha",
		"gpt-4-32k-0613":				"vizcacha",
		"claude-instant":               "a2",
		"claude-2-100k":                "a2_2",
		"claude-instant-100k":          "a2_100k",
		"google-palm":                  "acouchy",
		"llama_2_70b_chat":             "llama_2_70b_chat",
	}

	Models.Object = ""

	for key := range Conf.Bot {
		Models.Data = append(Models.Data, ModelDef{
			ID:      key,
			Object:  "",
			Created: 0,
			OwnedBy: "",
		})
	}
}
