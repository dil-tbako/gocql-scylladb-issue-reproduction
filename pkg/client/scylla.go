package client

import (
    "fmt"
    "time"
    
    "github.com/gocql/gocql"
)

type ScyllaClient struct {
    session *gocql.Session
}

func NewScyllaClient(hosts []string) (*ScyllaClient, error) {
    cluster := gocql.NewCluster(hosts...)
    cluster.Keyspace = "reproduction"
    cluster.Consistency = gocql.Quorum
    cluster.Timeout = 10 * time.Second
    cluster.ConnectTimeout = 10 * time.Second
    
    session, err := cluster.CreateSession()
    if err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    
    return &ScyllaClient{session: session}, nil
}

func (c *ScyllaClient) Close() {
    if c.session != nil {
        c.session.Close()
    }
}

func (c *ScyllaClient) InsertUser(orgID, userID, userData string, version int) error {
    query := `INSERT INTO platform_users (org_id, user_id, user_data, version, created_at) 
              VALUES (?, ?, ?, ?, toTimestamp(now()))`
    
    return c.session.Query(query, orgID, userID, userData, version).Exec()
}

func (c *ScyllaClient) DeleteUsersByOrgID(orgID string) error {
    query := `DELETE FROM platform_users WHERE org_id = ?`
    return c.session.Query(query, orgID).Exec()
}

func (c *ScyllaClient) CountUsersByOrgID(orgID string) (int, error) {
    query := `SELECT COUNT(*) FROM platform_users WHERE org_id = ?`
    
    var count int
    err := c.session.Query(query, orgID).Scan(&count)
    return count, err
}

func (c *ScyllaClient) GetAllUsersByOrgID(orgID string) ([]map[string]interface{}, error) {
    query := `SELECT user_id, user_data, version FROM platform_users WHERE org_id = ?`
    
    iter := c.session.Query(query, orgID).Iter()
    defer iter.Close()
    
    var users []map[string]interface{}
    var userID, userData string
    var version int
    
    for iter.Scan(&userID, &userData, &version) {
        users = append(users, map[string]interface{}{
            "user_id":   userID,
            "user_data": userData,
            "version":   version,
        })
    }
    
    return users, iter.Close()
}