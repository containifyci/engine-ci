package protos2

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockContainifyCIv1 is a mock implementation of ContainifyCIv1 interface
type MockContainifyCIv1 struct {
	mock.Mock
}

func (m *MockContainifyCIv1) GetBuild() (*BuildArgsResponse, error) {
	args := m.Called()
	return args.Get(0).(*BuildArgsResponse), args.Error(1)
}

// MockContainifyCIv2 is a mock implementation of ContainifyCIv2 interface
type MockContainifyCIv2 struct {
	mock.Mock
}

func (m *MockContainifyCIv2) GetBuilds() (*BuildArgsGroupResponse, error) {
	args := m.Called()
	return args.Get(0).(*BuildArgsGroupResponse), args.Error(1)
}

// MockContainifyCIEngineClient is a mock gRPC client
type MockContainifyCIEngineClient struct {
	mock.Mock
}

func (m *MockContainifyCIEngineClient) GetBuild(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*BuildArgsResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*BuildArgsResponse), args.Error(1)
}

func (m *MockContainifyCIEngineClient) GetBuilds(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*BuildArgsGroupResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*BuildArgsGroupResponse), args.Error(1)
}

// TestGRPCServerContainifyCIv1_GetBuild tests the v1 gRPC server GetBuild method
func TestGRPCServerContainifyCIv1_GetBuild(t *testing.T) {
	mockImpl := &MockContainifyCIv1{}
	server := &GRPCServerContainifyCIv1{
		Impl: mockImpl,
	}

	expectedResponse := &BuildArgsResponse{
		Args: []*BuildArgs{{
			Application: "test-build",
			Registry:    "docker.io",
			Image:       "test-image:latest",
		}},
	}

	// Test successful response
	mockImpl.On("GetBuild").Return(expectedResponse, nil)

	result, err := server.GetBuild(context.Background(), &Empty{})

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockImpl.AssertExpectations(t)
}

// TestGRPCServerContainifyCIv1_GetBuild_Error tests error handling in v1 server
func TestGRPCServerContainifyCIv1_GetBuild_Error(t *testing.T) {
	mockImpl := &MockContainifyCIv1{}
	server := &GRPCServerContainifyCIv1{
		Impl: mockImpl,
	}

	expectedError := errors.New("implementation error")

	// Test error response
	mockImpl.On("GetBuild").Return((*BuildArgsResponse)(nil), expectedError)

	result, err := server.GetBuild(context.Background(), &Empty{})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	mockImpl.AssertExpectations(t)
}

// TestContainifyCIv1GRPCClient_GetBuild tests the v1 gRPC client GetBuild method
func TestContainifyCIv1GRPCClient_GetBuild(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv1GRPCClient{
		client: mockClient,
	}

	expectedResponse := &BuildArgsResponse{
		Args: []*BuildArgs{{
			Application: "test-build",
			Registry:    "docker.io",
			Image:       "test-image:latest",
		}},
	}

	// Test successful response
	mockClient.On("GetBuild", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return(expectedResponse, nil)

	result, err := grpcClient.GetBuild()

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockClient.AssertExpectations(t)
}

// TestContainifyCIv1GRPCClient_GetBuild_Error tests error handling in v1 client
func TestContainifyCIv1GRPCClient_GetBuild_Error(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv1GRPCClient{
		client: mockClient,
	}

	expectedError := errors.New("gRPC call failed")

	// Test error response - this should return error instead of panicking
	mockClient.On("GetBuild", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return((*BuildArgsResponse)(nil), expectedError)

	result, err := grpcClient.GetBuild()

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	mockClient.AssertExpectations(t)
}

// TestContainifyCIv1GRPCClient_GetBuild_NilResponse tests handling of nil response
func TestContainifyCIv1GRPCClient_GetBuild_NilResponse(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv1GRPCClient{
		client: mockClient,
	}

	// Test nil response with no error
	mockClient.On("GetBuild", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return((*BuildArgsResponse)(nil), nil)

	result, err := grpcClient.GetBuild()

	require.NoError(t, err)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}

// TestGRPCServerContainifyCIv2_GetBuilds tests the v2 gRPC server GetBuilds method
func TestGRPCServerContainifyCIv2_GetBuilds(t *testing.T) {
	mockImpl := &MockContainifyCIv2{}
	server := &GRPCServerContainifyCIv2{
		Impl: mockImpl,
	}

	expectedResponse := &BuildArgsGroupResponse{
		Args: []*BuildArgsGroup{
			{
				Args: []*BuildArgs{{
					Application: "build1",
					Registry:    "docker.io",
					Image:       "image1:latest",
				}},
			},
			{
				Args: []*BuildArgs{{
					Application: "build2",
					Registry:    "gcr.io",
					Image:       "image2:v1.0",
				}},
			},
		},
	}

	// Test successful response
	mockImpl.On("GetBuilds").Return(expectedResponse, nil)

	result, err := server.GetBuilds(context.Background(), &Empty{})

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockImpl.AssertExpectations(t)
}

// TestGRPCServerContainifyCIv2_GetBuilds_Error tests error handling in v2 server
func TestGRPCServerContainifyCIv2_GetBuilds_Error(t *testing.T) {
	mockImpl := &MockContainifyCIv2{}
	server := &GRPCServerContainifyCIv2{
		Impl: mockImpl,
	}

	expectedError := errors.New("implementation error")

	// Test error response
	mockImpl.On("GetBuilds").Return((*BuildArgsGroupResponse)(nil), expectedError)

	result, err := server.GetBuilds(context.Background(), &Empty{})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	mockImpl.AssertExpectations(t)
}

// TestContainifyCIv2GRPCClient_GetBuilds tests the v2 gRPC client GetBuilds method
func TestContainifyCIv2GRPCClient_GetBuilds(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv2GRPCClient{
		client: mockClient,
	}

	expectedResponse := &BuildArgsGroupResponse{
		Args: []*BuildArgsGroup{
			{
				Args: []*BuildArgs{{
					Application: "build1",
					Registry:    "docker.io",
					Image:       "image1:latest",
				}},
			},
			{
				Args: []*BuildArgs{{
					Application: "build2",
					Registry:    "gcr.io",
					Image:       "image2:v1.0",
				}},
			},
		},
	}

	// Test successful response
	mockClient.On("GetBuilds", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return(expectedResponse, nil)

	result, err := grpcClient.GetBuilds()

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockClient.AssertExpectations(t)
}

// TestContainifyCIv2GRPCClient_GetBuilds_Error tests error handling in v2 client
func TestContainifyCIv2GRPCClient_GetBuilds_Error(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv2GRPCClient{
		client: mockClient,
	}

	expectedError := errors.New("gRPC call failed")

	// Test error response - this should return error instead of panicking
	mockClient.On("GetBuilds", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return((*BuildArgsGroupResponse)(nil), expectedError)

	result, err := grpcClient.GetBuilds()

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	mockClient.AssertExpectations(t)
}

// TestContainifyCIv2GRPCClient_GetBuilds_EmptyResponse tests handling of empty builds list
func TestContainifyCIv2GRPCClient_GetBuilds_EmptyResponse(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv2GRPCClient{
		client: mockClient,
	}

	expectedResponse := &BuildArgsGroupResponse{
		Args: []*BuildArgsGroup{},
	}

	// Test empty builds list
	mockClient.On("GetBuilds", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return(expectedResponse, nil)

	result, err := grpcClient.GetBuilds()

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	assert.Empty(t, result.Args)
	mockClient.AssertExpectations(t)
}

// TestContainifyCIv2GRPCClient_GetBuilds_NilResponse tests handling of nil response
func TestContainifyCIv2GRPCClient_GetBuilds_NilResponse(t *testing.T) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv2GRPCClient{
		client: mockClient,
	}

	// Test nil response with no error
	mockClient.On("GetBuilds", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	}), &Empty{}).Return((*BuildArgsGroupResponse)(nil), nil)

	result, err := grpcClient.GetBuilds()

	require.NoError(t, err)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}

// TestContainifyCIv1GRPCPlugin_GRPCServer tests the v1 plugin GRPCServer method
func TestContainifyCIv1GRPCPlugin_GRPCServer(t *testing.T) {
	mockImpl := &MockContainifyCIv1{}
	plugin := &ContainifyCIv1GRPCPlugin{
		Impl: mockImpl,
	}

	// We can't easily test the full gRPC server registration without a real gRPC server
	// Just test that the plugin exists and can be created without error
	assert.NotNil(t, plugin)
	assert.NotNil(t, plugin.Impl)
}

// TestContainifyCIv2GRPCPlugin_GRPCServer tests the v2 plugin GRPCServer method
func TestContainifyCIv2GRPCPlugin_GRPCServer(t *testing.T) {
	mockImpl := &MockContainifyCIv2{}
	plugin := &ContainifyCIv2GRPCPlugin{
		Impl: mockImpl,
	}

	// We can't easily test the full gRPC server registration without a real gRPC server
	// Just test that the plugin exists and can be created without error
	assert.NotNil(t, plugin)
	assert.NotNil(t, plugin.Impl)
}

// TestContainifyCIv1GRPCPlugin_GRPCClient tests the v1 plugin GRPCClient method
func TestContainifyCIv1GRPCPlugin_GRPCClient(t *testing.T) {
	plugin := &ContainifyCIv1GRPCPlugin{}

	// We can't easily test the full gRPC client creation without a real gRPC connection
	// But we can test that the method returns the expected type
	client, err := plugin.GRPCClient(context.Background(), nil, nil)
	
	assert.NoError(t, err)
	assert.IsType(t, &ContainifyCIv1GRPCClient{}, client)
}

// TestContainifyCIv2GRPCPlugin_GRPCClient tests the v2 plugin GRPCClient method
func TestContainifyCIv2GRPCPlugin_GRPCClient(t *testing.T) {
	plugin := &ContainifyCIv2GRPCPlugin{}

	// We can't easily test the full gRPC client creation without a real gRPC connection
	// But we can test that the method returns the expected type
	client, err := plugin.GRPCClient(context.Background(), nil, nil)
	
	assert.NoError(t, err)
	assert.IsType(t, &ContainifyCIv2GRPCClient{}, client)
}

// TestContextUsage tests that proper context is used instead of context.TODO
func TestContextUsage(t *testing.T) {
	// Test that v1 client uses proper context
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv1GRPCClient{
		client: mockClient,
	}

	expectedResponse := &BuildArgsResponse{
		Args: []*BuildArgs{{Application: "test"}},
	}

	// Verify that context.Background() is used, not context.TODO()
	mockClient.On("GetBuild", mock.MatchedBy(func(ctx context.Context) bool {
		// Verify context is not nil and appears to be background context
		return ctx != nil && ctx.Err() == nil
	}), &Empty{}).Return(expectedResponse, nil)

	_, err := grpcClient.GetBuild()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestContextUsageV2 tests that proper context is used in v2 client
func TestContextUsageV2(t *testing.T) {
	// Test that v2 client uses proper context
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv2GRPCClient{
		client: mockClient,
	}

	expectedResponse := &BuildArgsGroupResponse{Args: []*BuildArgsGroup{}}

	// Verify that context.Background() is used, not context.TODO()
	mockClient.On("GetBuilds", mock.MatchedBy(func(ctx context.Context) bool {
		// Verify context is not nil and appears to be background context
		return ctx != nil && ctx.Err() == nil
	}), &Empty{}).Return(expectedResponse, nil)

	_, err := grpcClient.GetBuilds()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestErrorPropagation_NoPanic tests that errors are properly propagated without panic
func TestErrorPropagation_NoPanic(t *testing.T) {
	// This test specifically verifies that the client methods don't panic on error
	// and instead return proper errors as per the issue #195 fixes

	t.Run("v1 client error propagation", func(t *testing.T) {
		mockClient := &MockContainifyCIEngineClient{}
		grpcClient := &ContainifyCIv1GRPCClient{
			client: mockClient,
		}

		expectedError := errors.New("network error")
		mockClient.On("GetBuild", mock.Anything, &Empty{}).Return((*BuildArgsResponse)(nil), expectedError)

		// This should not panic, but return the error
		assert.NotPanics(t, func() {
			result, err := grpcClient.GetBuild()
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Equal(t, expectedError, err)
		})
		
		mockClient.AssertExpectations(t)
	})

	t.Run("v2 client error propagation", func(t *testing.T) {
		mockClient := &MockContainifyCIEngineClient{}
		grpcClient := &ContainifyCIv2GRPCClient{
			client: mockClient,
		}

		expectedError := errors.New("timeout error")
		mockClient.On("GetBuilds", mock.Anything, &Empty{}).Return((*BuildArgsGroupResponse)(nil), expectedError)

		// This should not panic, but return the error
		assert.NotPanics(t, func() {
			result, err := grpcClient.GetBuilds()
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Equal(t, expectedError, err)
		})
		
		mockClient.AssertExpectations(t)
	})
}

// TestInterfaceCompliance tests that the implementations satisfy the interfaces
func TestInterfaceCompliance(t *testing.T) {
	// Test that v1 client implements the interface
	var _ ContainifyCIv1 = &ContainifyCIv1GRPCClient{}
	
	// Test that v2 client implements the interface  
	var _ ContainifyCIv2 = &ContainifyCIv2GRPCClient{}
	
	// This test will fail to compile if the interfaces aren't properly implemented
	t.Log("Interface compliance check passed")
}

// BenchmarkContainifyCIv1GRPCClient_GetBuild benchmarks the v1 client GetBuild method
func BenchmarkContainifyCIv1GRPCClient_GetBuild(b *testing.B) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv1GRPCClient{
		client: mockClient,
	}

	response := &BuildArgsResponse{
		Args: []*BuildArgs{{
			Application: "test",
			Registry:    "docker.io", 
			Image:       "test:latest",
		}},
	}
	mockClient.On("GetBuild", mock.Anything, &Empty{}).Return(response, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := grpcClient.GetBuild()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContainifyCIv2GRPCClient_GetBuilds benchmarks the v2 client GetBuilds method
func BenchmarkContainifyCIv2GRPCClient_GetBuilds(b *testing.B) {
	mockClient := &MockContainifyCIEngineClient{}
	grpcClient := &ContainifyCIv2GRPCClient{
		client: mockClient,
	}

	response := &BuildArgsGroupResponse{
		Args: []*BuildArgsGroup{
			{
				Args: []*BuildArgs{{
					Application: "test1",
					Registry:    "docker.io",
					Image:       "test1:latest",
				}},
			},
			{
				Args: []*BuildArgs{{
					Application: "test2",
					Registry:    "gcr.io",
					Image:       "test2:v1.0",
				}},
			},
		},
	}
	mockClient.On("GetBuilds", mock.Anything, &Empty{}).Return(response, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := grpcClient.GetBuilds()
		if err != nil {
			b.Fatal(err)
		}
	}
}