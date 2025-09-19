package tests

import (
    "fmt"
    "runtime"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/tbako/gocql-scylladb-issue-reproduction/pkg/client"
)

func TestDeleteByPartitionKey_ReproducesIssue(t *testing.T) {
    t.Logf("Running on platform: %s/%s", runtime.GOOS, runtime.GOARCH)
    
    // Connect to ScyllaDB
    scyllaClient, err := client.NewScyllaClient([]string{"127.0.0.1:9043"})
    require.NoError(t, err, "Failed to connect to ScyllaDB")
    defer scyllaClient.Close()
    
    orgID := fmt.Sprintf("test-org-%d", time.Now().Unix())
    
    t.Run("setup_test_data", func(t *testing.T) {
        // Insert multiple users for the same organization
        for i := 1; i <= 3; i++ {
            userID := fmt.Sprintf("user-%d", i)
            userData := fmt.Sprintf(`{"name":"User%d"}`, i)
            
            err := scyllaClient.InsertUser(orgID, userID, userData, 1)
            require.NoError(t, err, "Failed to insert user %s", userID)
        }
        
        // Verify data is inserted
        count, err := scyllaClient.CountUsersByOrgID(orgID)
        require.NoError(t, err, "Failed to count users")
        assert.Equal(t, 3, count, "Expected 3 users to be inserted")
        
        users, err := scyllaClient.GetAllUsersByOrgID(orgID)
        require.NoError(t, err, "Failed to get users")
        assert.Len(t, users, 3, "Expected 3 users in result set")
    })
    
    t.Run("delete_by_partition_key", func(t *testing.T) {
        // This is the operation that fails on certain platforms
        err := scyllaClient.DeleteUsersByOrgID(orgID)
        assert.NoError(t, err, "DELETE by partition key should not return error")
        
        // Verify deletion worked
        count, err := scyllaClient.CountUsersByOrgID(orgID)
        require.NoError(t, err, "Failed to count users after deletion")
        
        // This is where the issue manifests
        if count != 0 {
            t.Errorf("ISSUE REPRODUCED: DELETE by partition key failed silently")
            t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
            t.Logf("Expected 0 users, got %d users", count)
            
            // Show remaining users for debugging
            users, err := scyllaClient.GetAllUsersByOrgID(orgID)
            if err == nil {
                t.Logf("Remaining users: %+v", users)
            }
        } else {
            t.Logf("SUCCESS: DELETE by partition key worked correctly")
        }
        
        assert.Equal(t, 0, count, "All users should be deleted when deleting by partition key")
    })
}

func TestPlatformComparison(t *testing.T) {
    t.Logf("=== PLATFORM INFORMATION ===")
    t.Logf("GOOS: %s", runtime.GOOS)
    t.Logf("GOARCH: %s", runtime.GOARCH)
    t.Logf("Go Version: %s", runtime.Version())
    t.Logf("NumCPU: %d", runtime.NumCPU())
    
    // This test documents the platform where the test runs
    // and helps identify patterns in the CI results
}