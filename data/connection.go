package data

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/asendia/legacy-api/simple"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ConnectDB(ctx context.Context, connStrCfg DBConnStrConfig) (*pgxpool.Pool, error) {
	cfg, err := connStrCfg.GeneratePGXPoolConfig()
	if err != nil {
		return nil, err
	}
	if connStrCfg.PoolMaxConns != 0 {
		cfg.MaxConns = connStrCfg.PoolMaxConns
	}
	if connStrCfg.PoolMinConns != 0 {
		cfg.MinConns = connStrCfg.PoolMinConns
	}
	if connStrCfg.PoolMaxConnIdleTime != 0 {
		cfg.MaxConnIdleTime = connStrCfg.PoolMaxConnIdleTime
	}
	if connStrCfg.PoolMaxConnLifetime != 0 {
		cfg.MaxConnLifetime = connStrCfg.PoolMaxConnLifetime
	}
	if connStrCfg.PoolHealthCheckPeriod != 0 {
		cfg.HealthCheckPeriod = connStrCfg.PoolHealthCheckPeriod
	}
	conn, err := pgxpool.ConnectConfig(ctx, cfg)
	return conn, err
}

func LoadDBURLConfig() DBConnStrConfig {
	cfg := DBConnStrConfig{}
	cfg.Username = simple.DefaultString(os.Getenv("DB_USER"), cfg.Username)
	cfg.Password = simple.DefaultString(os.Getenv("DB_PASSWORD"), cfg.Password)
	cfg.Host = simple.DefaultString(os.Getenv("DB_HOST"), cfg.Host)
	cfg.Port = convertStrToIntFallback(os.Getenv("DB_PORT"), 5432)
	cfg.Database = simple.DefaultString(os.Getenv("DB_NAME"), cfg.Database)
	// Cloud Function with Cloud SQL
	cfg.SocketDir = simple.DefaultString(os.Getenv("DB_SOCKET_DIR"), "/cloudsql")
	cfg.InstanceConnectionName = os.Getenv("INSTANCE_CONNECTION_NAME")
	// pgx
	cfg.PoolMaxConns = int32(convertStrToIntFallback(os.Getenv("DB_MAX_CONNS"), 0))
	cfg.PoolMinConns = int32(convertStrToIntFallback(os.Getenv("DB_MIN_CONNS"), 0))
	cfg.PoolMaxConnLifetime = time.Second *
		time.Duration(convertStrToIntFallback(os.Getenv("DB_MAX_CONN_LIFETIME_SEC"), 0))
	cfg.PoolMaxConnIdleTime = time.Second *
		time.Duration(convertStrToIntFallback(os.Getenv("DB_MAX_CONN_IDLE_TIME_SEC"), 0))
	cfg.PoolHealthCheckPeriod = time.Second *
		time.Duration(convertStrToIntFallback(os.Getenv("DB_HEALTH_CHECK_PERIOD_SEC"), 0))
	return cfg
}

type DBConnStrConfig struct {
	Username string
	Password string
	Host     string
	Port     int
	Database string
	// https://cloud.google.com/sql/docs/postgres/connect-functions#go
	SocketDir              string
	InstanceConnectionName string
	// pgxPool
	PoolMaxConns          int32
	PoolMinConns          int32
	PoolMaxConnLifetime   time.Duration
	PoolMaxConnIdleTime   time.Duration
	PoolHealthCheckPeriod time.Duration
}

// return
// .e.g.
// local = "user=legacy_admin database=legacy host=localhost host=localhost port=5432"
// gcp   = "user=legacy_admin password=secure_password database=legacy host=/cloudsql/instance_connection_name"
func (d *DBConnStrConfig) GenerateConnString() string {
	s := "user=" + d.Username
	if d.Password != "" {
		s += " password=" + d.Password
	}
	s += " database=" + d.Database
	if d.InstanceConnectionName == "" || d.SocketDir == "" {
		s += " host=" + d.Host
	} else {
		s += " host=" + d.SocketDir + "/" + d.InstanceConnectionName
	}
	if d.Port != 0 {
		s += " port=" + fmt.Sprint(d.Port)
	}
	return s
}

func (d *DBConnStrConfig) GeneratePGXPoolConfig() (*pgxpool.Config, error) {
	connStr := d.GenerateConnString()
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func convertStrToIntFallback(s string, fallback int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return i
}
