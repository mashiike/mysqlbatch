package mysqlbatch_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/mashiike/mysqlbatch"
	"github.com/stretchr/testify/require"
)

func TestSSMParameterFetcher(t *testing.T) {
	var apiCallCount int32
	remotePassword := "test password"
	fetcher := &mysqlbatch.SSMParameterFetcher{
		LoadAWSDefaultConfigOptions: []func(*config.LoadOptions) error{
			config.WithRegion("ap-northeast-1"),
			config.WithAPIOptions([]func(stack *middleware.Stack) error{
				func(stack *middleware.Stack) error {
					return stack.Finalize.Add(
						middleware.FinalizeMiddlewareFunc("test",
							func(context.Context, middleware.FinalizeInput, middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
								atomic.AddInt32(&apiCallCount, 1)
								return middleware.FinalizeOutput{
									Result: &ssm.GetParameterOutput{
										Parameter: &types.Parameter{
											Value: &remotePassword,
										},
									},
								}, middleware.Metadata{}, nil
							},
						),
						middleware.Before,
					)
				},
			}),
		},
	}
	now := time.Now()
	restore := flextime.Fix(now)
	defer restore()
	actual, err := fetcher.Fetch(context.Background(), "/test/DBPASSWORD", "passowrd")
	require.NoError(t, err)
	require.Equal(t, "test password", actual)
	require.EqualValues(t, 1, apiCallCount)

	remotePassword = "test password2"
	flextime.Fix(now.Add(5 * time.Minute))
	actual, err = fetcher.Fetch(context.Background(), "/test/DBPASSWORD", "")
	require.NoError(t, err)
	require.Equal(t, "test password", actual)
	require.EqualValues(t, 1, apiCallCount)

	actual, err = fetcher.Fetch(context.Background(), "/test2/DBPASSWORD", "")
	require.NoError(t, err)
	require.Equal(t, "test password2", actual)
	require.EqualValues(t, 2, apiCallCount)

	flextime.Fix(now.Add(20 * time.Minute))
	actual, err = fetcher.Fetch(context.Background(), "/test/DBPASSWORD", "")
	require.NoError(t, err)
	require.Equal(t, "test password2", actual)
	require.EqualValues(t, int32(3), apiCallCount)
}

func TestSSMParameterFetcher__JSON(t *testing.T) {
	var apiCallCount int32
	remotePassword := "test password"
	fetcher := &mysqlbatch.SSMParameterFetcher{
		LoadAWSDefaultConfigOptions: []func(*config.LoadOptions) error{
			config.WithRegion("ap-northeast-1"),
			config.WithAPIOptions([]func(stack *middleware.Stack) error{
				func(stack *middleware.Stack) error {
					return stack.Finalize.Add(
						middleware.FinalizeMiddlewareFunc("test",
							func(context.Context, middleware.FinalizeInput, middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
								atomic.AddInt32(&apiCallCount, 1)
								return middleware.FinalizeOutput{
									Result: &ssm.GetParameterOutput{
										Parameter: &types.Parameter{
											Value: aws.String(
												fmt.Sprintf(`{"password":"%s", "name":"hoge", "port":3306}`, remotePassword),
											),
										},
									},
								}, middleware.Metadata{}, nil
							},
						),
						middleware.Before,
					)
				},
			}),
		},
	}
	now := time.Now()
	restore := flextime.Fix(now)
	defer restore()
	actual, err := fetcher.Fetch(context.Background(), "/test/DBPASSWORD", "password")
	require.NoError(t, err)
	require.Equal(t, "test password", actual)
	require.EqualValues(t, 1, apiCallCount)

	remotePassword = "test password2"
	flextime.Fix(now.Add(5 * time.Minute))
	actual, err = fetcher.Fetch(context.Background(), "/test/DBPASSWORD", "password")
	require.NoError(t, err)
	require.Equal(t, "test password", actual)
	require.EqualValues(t, 1, apiCallCount)

	flextime.Fix(now.Add(20 * time.Minute))
	actual, err = fetcher.Fetch(context.Background(), "/test/DBPASSWORD", "password")
	require.NoError(t, err)
	require.Equal(t, "test password2", actual)
	require.EqualValues(t, int32(2), apiCallCount)
}
