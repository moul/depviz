package sql // import "moul.io/depviz/sql"
import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"moul.io/depviz/model"
	"moul.io/zapgorm"
)

func FromOpts(opts *Options) (*gorm.DB, error) {
	if os.Getenv("DEPVIZ_DEBUG") == "1" {
		opts.Verbose = true
	}
	// configure sql
	var (
		db  *gorm.DB
		err error
	)
	switch {
	case strings.HasPrefix(opts.Config, "sqlite://"):
		dbPath := os.ExpandEnv(opts.Config[len("sqlite://"):])
		db, err = gorm.Open("sqlite3", dbPath)
	default:
		return nil, fmt.Errorf("unsupported sql driver: %q", opts.Config)
	}
	if err != nil {
		return nil, err
	}
	db.LogMode(true)
	log.SetOutput(ioutil.Discard)
	db.Callback().Create().Remove("gorm:update_time_stamp")
	db.Callback().Update().Remove("gorm:update_time_stamp")
	log.SetOutput(os.Stderr)
	db.SetLogger(zapgorm.New(zap.L().Named("vendor.gorm")))
	db = db.Set("gorm:auto_preload", true)
	db = db.Set("gorm:association_autoupdate", true)
	db.BlockGlobalUpdate(true)
	db.SingularTable(true)
	db.LogMode(opts.Verbose)
	if err := db.AutoMigrate(model.AllModels...).Error; err != nil {
		return nil, err
	}

	return db, nil
}
