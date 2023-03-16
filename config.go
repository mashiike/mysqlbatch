package mysqlbatch

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Songmu/flextime"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

// Config is a connection setting to MySQL.
// Exists to generate a Golang connection DSN to MySQL
type Config struct {
	DSN      string
	User     string
	Password string
	Host     string
	Port     int
	Database string

	PasswordSSMParameterName string
	Fetcher                  *SSMParameterFetcher
}

type SSMParameterFetcher struct {
	LoadAWSDefaultConfigOptions []func(*config.LoadOptions) error
	ssmClient                   *ssm.Client
	mu                          sync.RWMutex
	g                           singleflight.Group
	fetchedAt                   map[string]time.Time
	cachedValue                 map[string]string
}

// NewDefaultConfig returns the config for connecting to the local MySQL server
func NewDefaultConfig() *Config {
	return &Config{
		User:    "root",
		Host:    "127.0.0.1",
		Port:    3306,
		Fetcher: &SSMParameterFetcher{},
	}
}

// GetDSN returns a DSN dedicated to connecting to MySQL.
func (c *Config) GetDSN(ctx context.Context) (string, error) {
	if c.DSN != "" {
		return strings.TrimPrefix(c.DSN, "mysql://"), nil
	}
	password := c.Password
	if c.needToRetrievePasswordRemotely() {
		var err error
		password, err = c.Fetcher.Fetch(ctx, c.PasswordSSMParameterName)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		c.User,
		password,
		c.Host,
		c.Port,
		c.Database,
	), nil
}

func (c *Config) needToRetrievePasswordRemotely() bool {
	return c.Password == "" && c.PasswordSSMParameterName != ""
}

func (f *SSMParameterFetcher) Fetch(ctx context.Context, parameterName string) (string, error) {

	if password, ok := f.fetchFromCache(parameterName); ok {
		return password, nil
	}
	password, err := f.fetchFromRemote(ctx, parameterName)
	if err != nil {
		return "", fmt.Errorf("getFromRemote: %w", err)
	}
	return password, nil
}

func (f *SSMParameterFetcher) fetchFromRemote(ctx context.Context, parameterName string) (string, error) {
	v, err, _ := f.g.Do("fetchRemote", func() (interface{}, error) {
		if f.ssmClient == nil {
			awsConf, err := config.LoadDefaultConfig(ctx, f.LoadAWSDefaultConfigOptions...)
			if err != nil {
				return nil, err
			}
			f.ssmClient = ssm.NewFromConfig(awsConf)
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		log.Printf("get ssm parameter `%s`", parameterName)
		output, err := f.ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           aws.String(parameterName),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return nil, err
		}
		f.setToCache(parameterName, *output.Parameter.Value)
		return *output.Parameter.Value, nil
	})
	if err != nil {
		return "", err
	}
	if password, ok := v.(string); ok {
		return password, nil
	}
	return "", errors.New("v is not string")
}

var cacheTTL time.Duration = 15 * time.Minute

func (f *SSMParameterFetcher) fetchFromCache(parameterName string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.fetchedAt == nil || f.fetchedAt[parameterName].IsZero() {
		return "", false
	}
	if flextime.Since(f.fetchedAt[parameterName]) < cacheTTL {
		if f.cachedValue == nil {
			return "", false
		}
		password, ok := f.cachedValue[parameterName]
		return password, ok
	}
	return "", false
}

func (f *SSMParameterFetcher) setToCache(parameterName string, password string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fetchedAt == nil {
		f.fetchedAt = make(map[string]time.Time)
	}
	f.fetchedAt[parameterName] = flextime.Now()
	if f.cachedValue == nil {
		f.cachedValue = make(map[string]string)
	}
	f.cachedValue[parameterName] = password
	log.Printf("cached ssm parameter `%s`,expire is %s", parameterName, f.fetchedAt[parameterName].Add(cacheTTL).Format(time.RFC3339))
	for key, t := range f.fetchedAt {
		if flextime.Since(t) >= cacheTTL {
			delete(f.fetchedAt, key)
			delete(f.cachedValue, key)
		}
	}
}
