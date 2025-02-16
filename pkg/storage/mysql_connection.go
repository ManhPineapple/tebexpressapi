package storage

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

type MysqlConfiguration struct {
	Port      int
	Host      string
	Username  string
	Password  string
	Database  string
	Charset   string
	ParseTime bool
	Loc       string
	Debug     bool
}

func DefaultMysqlFromConfig(opts *MysqlConfiguration) *MysqlConfiguration {
	if opts == nil {
		opts = &MysqlConfiguration{
			Host:      viper.GetString("mysql.host"),
			Port:      viper.GetInt("mysql.port"),
			Username:  viper.GetString("mysql.username"),
			Password:  viper.GetString("mysql.password"),
			Charset:   viper.GetString("mysql.charset"),
			Database:  viper.GetString("mysql.database"),
			ParseTime: viper.GetBool("mysql.parse_time"),
			Debug:     viper.GetBool("mysql.debug"),
		}
	}

	return opts
}

func NewMysqlConnection(opts *MysqlConfiguration) (*gorm.DB, error) {
	log.Println("PARAM: ", buildMysqlConnectionParam(opts))
	dsn := buildMysqlConnectionParam(opts)

	var newLogger gorm_logger.Interface

	if !opts.Debug {
		newLogger = gorm_logger.Default.LogMode(gorm_logger.Silent)
	} else {
		newLogger = gorm_logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			gorm_logger.Config{
				SlowThreshold:             time.Second,      // Slow SQL threshold
				LogLevel:                  gorm_logger.Info, // Log level
				IgnoreRecordNotFoundError: true,             // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,            // Disable color
			},
		)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Printf("Open mysql connection error, %v", err)
		return nil, err
	}

	return db, nil
}

func buildMysqlConnectionParam(opts *MysqlConfiguration) string {
	param := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s",
		opts.Username,
		opts.Password,
		opts.Host,
		opts.Port,
		opts.Database,
	)

	// Additional params
	additionParams := make(map[string]string)

	if len(opts.Charset) > 0 {
		additionParams["charset"] = opts.Charset
	}

	switch opts.ParseTime {
	case true:
		additionParams["parseTime"] = "True"
	case false:
		additionParams["parseTime"] = "False"
	}

	if len(opts.Loc) > 0 {
		additionParams["loc"] = opts.Loc
	}

	additionParamsStr := make([]string, 0)
	for k, v := range additionParams {
		additionParamsStr = append(additionParamsStr, fmt.Sprintf("%s=%s", k, v))
	}

	return fmt.Sprintf("%s?%s", param, strings.Join(additionParamsStr, "&"))
}
